package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
	"unsafe"

	"gcli/internal/config"
	"gcli/internal/parser"
)

type Result struct {
	Raw    []byte
	IsJson bool
}

func Init(putRes func(*Result)) {
	for i := 0; i < 10; i++ {
		putRes(&Result{})
	}
}

func DoHTTP(c *config.HTTPConfig, resObj *Result) error {
	fmt.Println("DoHTTP")

	const op = "transport.DoHTTP"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := prepareRequest(c, ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	res, err := clientDo(req, false)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer res.Body.Close()

	resObj.Raw, resObj.IsJson, err = readBody(res.Body, res)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func prepareRequest(c *config.HTTPConfig, ctx context.Context) (*http.Request, error) {
	const op = "transport.prepareRequest"

	if c == nil {
		return nil, fmt.Errorf("%s: nil config", op)
	}

	mtd := unsafe.String(unsafe.SliceData(c.Method), len(c.Method))
	url := unsafe.String(unsafe.SliceData(c.Url), len(c.Url))

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

func clientDo(req *http.Request, ic bool) (*http.Response, error) {
	const op = "transport.clientDo"

	cl := &http.Client{}

	if ic {
		cl.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return cl.Do(req)
}

func readBody(body io.ReadCloser, res *http.Response) ([]byte, bool, error) {
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
