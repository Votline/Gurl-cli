package transport

import (
	"io"
	"bytes"
	"strings"
	"net/http"
	"encoding/json"

	"Gurl-cli/internal/config"
)

type Result struct {
	Raw *http.Response
	JSON map[string]interface{}
}

func convData(res *http.Response) (map[string]interface{}, error) {
	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil, nil
	}

	var data map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&data)
	return data, err
}

func Get(url string) (Result, error) {
	res, err := http.Get(url)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	data, err := convData(res)
	if err != nil {
		return Result{Raw: res, JSON: nil}, nil
	}
	return Result{Raw: res, JSON: data}, nil
}

func Post(cfg *config.HTTPConfig) (Result, error) {
	var body io.Reader

	if cfg.Body != nil {
		jsonBytes, err := json.Marshal(cfg.Body)
		if err != nil {return Result{}, err}
		body = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(cfg.Method, cfg.Url, body)
	if err != nil {return Result{}, err}

	for header, value := range cfg.Headers {
		req.Header.Set(header, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	data, err := convData(res)
	if err != nil {
		return Result{Raw: res, JSON: nil}, nil
	}
	return Result{Raw: res, JSON: data}, nil
}
