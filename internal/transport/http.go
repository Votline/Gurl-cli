package transport

import (
	"io"
	"log"
	"bytes"
	"strings"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"Gurl-cli/internal/config"
)

type Result struct {
	Raw *http.Response
	RawBody []byte
	JSON map[string]interface{}
}

func convData(body []byte, res *http.Response) map[string]interface{} {
	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil
	}

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		log.Printf("JSON decoding error: %v", err.Error())
	}
	return data
}

func extBody(res *http.Response) []byte {
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Body reading error: %v", err.Error())
		return nil
	}
	return body
}

func Get(url string) (Result, error) {
	res, err := http.Get(url)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res)
	data := convData(body, res)

	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func Post(cfg *config.HTTPConfig) (Result, error) {
	var bodyReader io.Reader

	if cfg.Body != nil {
		jsonBytes, err := json.Marshal(cfg.Body)
		if err != nil {return Result{}, err}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(cfg.Method, cfg.Url, bodyReader)
	if err != nil {return Result{}, err}

	for header, value := range cfg.Headers {
		req.Header.Set(header, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res)
	data := convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func Del(cfg *config.HTTPConfig) (Result, error) {
	var bodyReader io.Reader

	if cfg.Body != nil {
		jsonBytes, err := json.Marshal(cfg.Body)
		if err != nil {return Result{}, err}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(cfg.Method, cfg.Url, bodyReader)
	if err != nil {return Result{}, err}

	for header, value := range cfg.Headers {
		req.Header.Set(header, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	body := extBody(res)
	data := convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}
