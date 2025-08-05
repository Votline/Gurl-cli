package config

import (
	"bytes"
	"errors"
	"strings"
	"strconv"
)

func findSource[T Config] (sourceCfg *T, cfgs []T, id string) bool {
	for _, cfg := range cfgs {
		if cfg.GetID() == id {
			*sourceCfg = cfg
			return true
		}
	}
	return false
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
		return nil, errors.New("RESPONSE template not found or invalid")
	}

	template := string(data[startIdx : endIdx+startIdx+1])
	parts := strings.SplitN(template, " ", 3)
	if len(parts) < 3 {
		return nil, errors.New("Invalid RESPONSE template.")
	}

	idPart := strings.TrimPrefix(parts[1], "id=")
	_, err := strconv.Atoi(idPart)
	if err != nil {
		return nil, errors.New("Invalid id in RESPONSE template.")
	}

	var sourceCfg T
	if !findSource(&sourceCfg, cfgs, idPart) {
		return nil, errors.New("Config not found. ID: " + idPart)
	}

	sourceResponse := sourceCfg.GetResponse()
	if sourceResponse == "" {
		return nil, errors.New("Config response is nil")
	}

	result := make([]byte, len(data)-(len(sourceResponse)+endIdx+1) )
	result = append(result, data[:startIdx-1]...)
	result = append(result, sourceResponse...)
	result = append(result, data[startIdx+endIdx+1:]...)
	return result, nil
}

func parseField[T Config](data []byte, cfgs []T) ([]byte, error) {
	if data == nil {
		return nil, errors.New("Config data nil")
	}
	newData, err := parse(data, cfgs)
	if err != nil {
		return nil, err
	}
	return newData, nil
}

func Parsing[T Config](cfg T, cfgs []T) (T, error) {
	if cfg.GetBody() != nil {
		newBody, err := parseField(cfg.GetBody(), cfgs)
		if err != nil {
			var zero T
			return zero, err
		}
		cfg.SetBody(newBody)
	}
	if headers, _ := cfg.GetHeaders(); headers != nil {
		hdrs, err := cfg.GetHeaders()
		if err != nil {
			var zero T
			return zero, err
		}
		newHeaders, err := parseField(hdrs, cfgs)
		if err != nil {
			var zero T
			return zero, err
		}
		cfg.SetHeaders(newHeaders)
	}
	return cfg, nil
}
