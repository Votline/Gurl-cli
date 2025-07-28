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

func Get(cfg *config.HTTPConfig) (Result, error) {
	req, err := prepareRequest(cfg)
	if err != nil {return Result{}, err}

	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res.Body)
	data := convData(body, res)

	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func Post(cfg *config.HTTPConfig) (Result, error) {
	req, err := prepareRequest(cfg)
	if err != nil {return Result{}, err}

	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res.Body)
	data := convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func Put(cfg *config.HTTPConfig) (Result, error) {
	req, err := prepareRequest(cfg)
	if err != nil {return Result{}, err}

	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res.Body)
	data := convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func Del(cfg *config.HTTPConfig) (Result, error) {
	req, err := prepareRequest(cfg)
	if err != nil {return Result{}, err}
	
	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res.Body)
	data := convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}
