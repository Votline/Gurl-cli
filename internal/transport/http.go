package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
	"unsafe"

	"gcli/internal/config"
	"gcli/internal/parser"
)

type Result struct {
	IsJSON bool
	Raw    []byte
	Cookie []byte
}
type Transport struct {
	jar *cookiejar.Jar
	cl  *http.Client
}

func NewTransport(putRes func(*Result)) *Transport {
	for i := 0; i < 10; i++ {
		putRes(&Result{})
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &Transport{jar: jar, cl: client}
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

	if c.HasFlag(config.FlagUseFileCookies) {
		jar, _ := cookiejar.New(nil)
		jar.SetCookies(req.URL, parser.UnparseCookies(c.GetCookie()))
		t.cl.Jar = jar
	}

	res, err := t.cl.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", op, err)
	}

	return res, nil
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
