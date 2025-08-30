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

func extBody(resBody io.ReadCloser) []byte {
	body, err := ioutil.ReadAll(resBody)
	if err != nil {
		log.Printf("Body reading error: %v", err.Error())
		return nil
	}
	return body
}

func prepareBody(body interface{}) (io.Reader, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			log.Printf("Marshal body error: %v", err)
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}
	return bodyReader, nil
}

func prepareRequest(cfg *config.HTTPConfig) (*http.Request, error) {
	bodyReader, err := prepareBody(cfg.Body)
	if err != nil {
		log.Printf("Prepare body err: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(cfg.Method, cfg.Url, bodyReader)
	if err != nil {
		log.Printf("Create request error: %v", err)
		return nil, err
	}

	for header, value := range cfg.Headers {
		req.Header.Set(header, value)
	}

	return req, nil
}
