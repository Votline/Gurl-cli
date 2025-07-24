package config

import (
	"os"
	"errors"
	"encoding/json"
)

type rawConfig struct {
	Type string `json:"type"`
}

func Decode(cfgPath string) (interface{}, error) {
	data, err := os.ReadFile(cfgPath)
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
