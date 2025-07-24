package config

import (
	"os"
	"errors"
	"encoding/json"
	"path/filepath"
)

type rawConfig struct {
	Type string `json:"type"`
}

func findConfigPath(userPath string) (string, error) {
	if _, err := os.Stat(userPath); err == nil {
		return userPath, nil
	}

	corePath, _ := os.Getwd()
	possiblePath := filepath.Join(corePath, userPath)
	if _, err := os.Stat(possiblePath); err == nil {
		return possiblePath, nil
	}

	msg := userPath + "\n" + possiblePath
	return "", errors.New("config not found in:\n" + msg)
}

func Decode(cfgPath string) (interface{}, error) {
	path, err := findConfigPath(cfgPath)
	if err != nil {return nil, err}

	data, err := os.ReadFile(path)
	if err != nil {return nil, err}
	
	var rawCfg rawConfig
	if err := json.Unmarshal(data, &rawCfg); err != nil {
		return nil, err
	}

	var cfg interface{}
	switch rawCfg.Type {
	case "http":
		cfg = &HTTPConfig{}
	case "grpc":
		cfg = &GRPCConfig{}
	default:
		return nil, errors.New("Unknown config type")
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
