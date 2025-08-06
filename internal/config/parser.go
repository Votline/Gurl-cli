package config

import (
	"bytes"
	"errors"
	"strings"
	"strconv"
)

func removeJsonShit(s string) string {
	s = strings.NewReplacer(
		`"`, "",
		`\`, "",
		`{`, "",
		`}`, "",
	).Replace(s)
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func findSource[T Config] (sourceCfg *T, cfgs []T, id string) bool {
	for _, cfg := range cfgs {
		if cfg.GetID() == id {
			*sourceCfg = cfg
			return true
		}
	}
	return false
}

func findID(data []byte, startIdx, endIdx int) (string, error) {
	template := string(data[startIdx : endIdx+startIdx+1])
	parts := strings.SplitN(template, " ", 3)
	if len(parts) < 3 {
		return "", errors.New("Invalid RESPONSE template")
	}

	idPart := strings.TrimPrefix(parts[1], "id=")
	_, err := strconv.Atoi(idPart)
	if err != nil {
		return "", errors.New("Invalid id in RESPONSE template")
	}

	return idPart, nil
}

func findIdx(data []byte) (startIdx, endIdx int) {
	startIdx = bytes.Index(data, []byte("RESPONSE id="))
	if startIdx == -1 {return -1, -1}
	endIdx = bytes.Index(data[startIdx:], []byte("}"))
	return
}

func parse[T Config](data []byte, cfgs []T) ([]byte, error) {
	startIdx, endIdx := findIdx(data)
	if startIdx == -1 || endIdx == -1 {
		return nil, nil
	}

	idPart, err := findID(data, startIdx, endIdx)
	if err != nil {return nil, err}

	var sourceCfg T
	if !findSource(&sourceCfg, cfgs, idPart) {
		return nil, errors.New("Config not found. ID: " + idPart)
	}

	sourceResponse := sourceCfg.GetResponse()
	if sourceResponse == "" {
		return nil, errors.New("Config response is nil")
	}

	response := removeJsonShit(sourceResponse)

	var result bytes.Buffer
	result.Write(data[:startIdx-1])
	result.WriteString(response)
	result.Write(data[startIdx+endIdx+1:])

	return result.Bytes(), nil
}

func Parsing[T Config](cfg T, cfgs []T) (T, error) {
	var zero T

	headers, err := cfg.GetHeaders()
	if err != nil {return zero, err}
	if headers != nil {
		newHeaders, err := parse(headers, cfgs)
		if err != nil {
			return zero, err
		}
		if newHeaders != nil {
			cfg.SetHeaders(newHeaders)
		}
	}

	body := cfg.GetBody()
	if body != nil {
		newBody, err := parse(body, cfgs)
		if err != nil {
			return zero, err
		}
		if newBody != nil {
			cfg.SetBody(newBody)
		}
	}

	return cfg, nil
}
