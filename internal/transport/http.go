package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	"gcli/internal/config"
	"gcli/internal/parser"
)

var builderPool = sync.Pool{
	New: func() any {
		return new(strings.Builder)
	},
}

type Result struct {
	Status int
	IsJSON bool
	Raw    []byte
	Cookie []byte
}
type Transport struct {
	jar map[string]string
	cl  *http.Client
}

func NewTransport(putRes func(*Result)) *Transport {
	for i := 0; i < 10; i++ {
		putRes(&Result{})
	}
	client := &http.Client{
		Jar: nil,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &Transport{jar: map[string]string{}, cl: client}
}

func (t *Transport) DoHTTP(c *config.HTTPConfig, resObj *Result) error {
	const op = "transport.DoHTTP"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := t.prepareRequest(c, ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	res, err := t.clientDo(req, c, false)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer res.Body.Close()

	resObj.Raw, resObj.IsJSON, err = t.readBody(res.Body, res)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	resObj.Cookie = parser.ParseCookies(req.URL, res.Cookies())

	resObj.Status = res.StatusCode

	return nil
}

func (t *Transport) prepareRequest(c *config.HTTPConfig, ctx context.Context) (*http.Request, error) {
	const op = "transport.prepareRequest"

	if c == nil {
		return nil, fmt.Errorf("%s: nil config", op)
	}

	mtd := unsafe.String(unsafe.SliceData(c.Method), len(c.Method))
	url := unsafe.String(unsafe.SliceData(c.URL), len(c.URL))

	var bRdr io.Reader
	if c.Body != nil {
		bRdr = bytes.NewReader(c.Body)
		ct := unsafe.String(unsafe.SliceData(c.Headers), len(c.Headers))
		parser.ParseContentType(&ct)
		if ct != "application/json" {
			bd := parser.ParseBody(c.Body)
			bRdr = bytes.NewReader(bd)
		}
	}

	req, err := http.NewRequestWithContext(ctx, mtd, url, bRdr)
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", op, err)
	}

	parser.ParseHeaders(c.Headers, func(k, v []byte) {
		key := unsafe.String(unsafe.SliceData(k), len(k))
		val := unsafe.String(unsafe.SliceData(v), len(v))
		req.Header.Set(key, val)
	})

	return req, nil
}

func (t *Transport) clientDo(req *http.Request, c *config.HTTPConfig, ic bool) (*http.Response, error) {
	const op = "transport.clientDo"
	if ic {
		t.cl.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	if len(t.jar) > 0 {
		sb := builderPool.Get().(*strings.Builder)
		sb.Reset()
		for k, v := range t.jar {
			if sb.Len() > 0 {
				sb.WriteByte(';')
			}
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(v)
		}
		req.Header.Set("Cookie", sb.String())
		builderPool.Put(sb)
	}

	if c.HasFlag(config.FlagUseFileCookies) {
		req.Header.Set("Cookie", "")
		parser.UnparseCookies(c.GetCookie(), func(ck string) {
			if req.Header.Get("Cookie") != "" {
				req.Header.Set("Cookie", req.Header.Get("Cookie")+"; "+ck)
			} else {
				req.Header.Set("Cookie", ck)
			}
		})
	}

	res, err := t.cl.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", op, err)
	}

	t.updateJar(res.Header["Set-Cookie"])

	return res, nil
}

func (t *Transport) updateJar(cookies []string) {
	const op = "transport.updateJar"

	for _, c := range cookies {
		if len(c) == 0 {
			continue
		}

		semi := strings.IndexByte(c, ';')
		if semi != -1 {
			c = c[:semi]
		}

		k, v, found := strings.Cut(c, "=")
		if !found {
			continue
		}

		t.jar[k] = v
	}
}

func (t *Transport) readBody(body io.ReadCloser, res *http.Response) ([]byte, bool, error) {
	const op = "transport.readBody"

	b, err := io.ReadAll(body)
	if err != nil {
		return nil, false, fmt.Errorf("%s: read body: %w", op, err)
	}

	ct := res.Header.Get("Content-Type")
	if len(ct) == 0 {
		ct = res.Header.Get("content-type")
	}
	if len(ct) == 0 {
		return b, false, nil
	}

	parser.ParseContentType(&ct)
	if ct == "" {
		return b, false, nil
	}

	return b, true, nil
}
