package transport

import (
	"strings"
	"net/http"
	"encoding/json"
)

type Result struct {
	Raw *http.Response
	JSON map[string]interface{}
}

func Get(url string) (Result, error){
	res, err := http.Get(url)
	if err != nil {return Result{}, err}
	defer res.Body.Close()

	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return Result{Raw: res, JSON: nil}, nil
	}

	var data map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil{
		return Result{Raw: res, JSON: nil}, nil
	}
	return Result{Raw: res, JSON: data}, nil
}
