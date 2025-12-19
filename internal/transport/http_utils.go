package transport

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"Gurl-cli/internal/config"
)

func (c *HTTPClient) convData(body []byte, res *http.Response) map[string]any {
	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil
	}

	var data map[string]any
	err := json.Unmarshal(body, &data)
	if err != nil {
		c.log.Error("JSON decoding error", zap.Error(err))
	}
	return data
}

func (c *HTTPClient) extBody(resBody io.ReadCloser) []byte {
	body, err := io.ReadAll(resBody)
	if err != nil {
		c.log.Error("Body reading error", zap.Error(err))
		return nil
	}
	return body
}

func (c *HTTPClient) prepareBody(body any, contentType string) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	var bodyReader io.Reader

	if strings.Contains(strings.ToLower(contentType), "application/json") {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			c.log.Error("Marshal body error", zap.Error(err))
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	} else {
		switch v := body.(type) {
		case json.RawMessage:

			var str string
			if err := json.Unmarshal(v, &str); err != nil {
				c.log.Error("Failed to unmarshal RawMessage", zap.Error(err))
				return nil, err
			}

			bodyReader = strings.NewReader(str)
		default:
			str := fmt.Sprintf("%v", v)
			bodyReader = strings.NewReader(str)
		}
	}
	return bodyReader, nil
}


func (c *HTTPClient) prepareRequest(cfg *config.HTTPConfig) (*http.Request, error) {
	contentType := "application/json"
	if ct, ok := cfg.Headers["Content-Type"]; ok {
		contentType = ct
	}

	bodyReader, err := c.prepareBody(cfg.Body, contentType)
	if err != nil {
		c.log.Error("Prepare body err", zap.Error(err))
		return nil, err
	}

	var bodyBytes []byte
	if bodyReader != nil {
		bodyBytes, _ = io.ReadAll(bodyReader)
	
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(cfg.Method, cfg.Url, bodyReader)
	if err != nil {
		c.log.Error("Create request error", zap.Error(err))
		return nil, err
	}

	for header, value := range cfg.Headers {
		req.Header.Set(header, value)
	}

	return req, nil
}

func (c *HTTPClient) clientDo(req *http.Request) (*http.Response, error) {
	cl := &http.Client{}
	jar := c.CkCl.GetJar()
	cl.Jar = jar

	if c.ic {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		cl.Transport = tr
	}

	res, err := cl.Do(req)
	if err != nil {
		c.log.Error("Do request error", zap.Error(err))
		return nil, err
	}
	if c.CkCl != nil {
		c.CkCl.UpdateCookies(req.URL)
	}
	return res, nil
}
