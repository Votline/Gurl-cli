package config

import (
	"bytes"
	"errors"
	"strings"
	"strconv"
)

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

	var found bool
	var sourceCfg T
	for _, cfg := range cfgs {
		if cfg.GetID() == idPart {
			sourceCfg = cfg
			found = true
		}
	}
	if !found {
		return nil, errors.New("Config not found. ID: " + idPart)
	}

	sourceResponse := sourceCfg.GetResponse()
	if sourceResponse == "" {
		return nil, errors.New("Config response is nil")
	}

	return []byte(sourceResponse), nil
}

func Parsing[T Config](cfg T, cfgs []T) (T, error) {
	if cfg.GetBody() != nil {
		newBody, err := parse(cfg.GetBody(), cfgs)
		if err != nil {
			var zero T
			return zero, err
		}
		cfg.SetBody(newBody)
		return cfg, nil
	}
	var zero T
	return zero, nil
}
