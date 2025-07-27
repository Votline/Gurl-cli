package transport

import (
	"net/http"

	"Gurl-cli/internal/config"
)

type Result struct {
	Raw *http.Response
	RawBody []byte
	JSON map[string]interface{}
}

func Get(url string) (Result, error) {
	res, err := http.Get(url)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res.Body)
	data := convData(body, res)

	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func Post(cfg *config.HTTPConfig) (Result, error) {
	bodyReader, err := prepareBody(cfg.Body)
	if err != nil {return Result{}, err}

	req, err := http.NewRequest(cfg.Method, cfg.Url, bodyReader)
	if err != nil {return Result{}, err}

	for header, value := range cfg.Headers {
		req.Header.Set(header, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res.Body)
	data := convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func Del(cfg *config.HTTPConfig) (Result, error) {
	bodyReader, err := prepareBody(cfg.Body)
	if err != nil {return Result{}, err}

	req, err := http.NewRequest(cfg.Method, cfg.Url, bodyReader)
	if err != nil {return Result{}, err}

	for header, value := range cfg.Headers {
		req.Header.Set(header, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res.Body)
	data := convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}
