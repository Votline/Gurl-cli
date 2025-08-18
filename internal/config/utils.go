package config

import (
	"os"
	"fmt"
	"log"
	"errors"
	"encoding/json"
	"path/filepath"
)

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

func parseTypedConfig[T Config](rawCfg []byte) (T, error) {
	var zero T
	var cfg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawCfg, &cfg); err != nil {
		log.Printf("Error unmarshalling config: %v\nSource: %v",
			err, string(rawCfg))
		return zero, err
	}

	var result interface{}
	switch cfg.Type {
	case "http":
		result = &HTTPConfig{}
	case "grpc":
		result = &GRPCConfig{}
	default:
		log.Printf("Invalid config type: %v", cfg.Type)
		return zero, errors.New("Invalid config type")
	}

	if err := json.Unmarshal(rawCfg, result); err != nil {
		log.Printf("Error unmarshalling config into result: %v\nSource: %v",
			err, string(rawCfg))
	}

	return result.(T), nil
}

func Decode[T Config](cfgPath string) ([]T, error) {
	path, err := findConfigPath(cfgPath)
	if err != nil {
		log.Printf("FindConfigPath error: %v", err)
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("ReadFile error: %v", err)
		return nil, err
	}
	
	var rawConfigs []json.RawMessage
	if err := json.Unmarshal(data, &rawConfigs); err == nil {
		configs := make([]T, len(rawConfigs))
		for i, rawCfg := range rawConfigs {
			cfg, err := parseTypedConfig[T](rawCfg)
			if err != nil {
				log.Printf("ParseTypedConfig multi error: %v", err)
				return nil, fmt.Errorf("config: %d: %v", i, err)
			}
			configs[i] = cfg
		}
		return configs, nil
	}
	
	cfg, err := parseTypedConfig[T](data)
	if err != nil {
		log.Printf("ParseTypedConfig solo error: %v", err)
		return nil, err
	}
	return []T{cfg}, nil
}

func ConfigUpd[T Config](parsed T, cfgPath string) error {
	cfgs, err := Decode[T](cfgPath)
	if err != nil {
		log.Printf("Decode config error: %v", err)
		return err
	}

	cfgID := parsed.GetID()
	for _, c := range cfgs {
		if c.GetID() == cfgID {
			c.SetResponse(parsed.GetResponse())
			break
		}
	}

	jsonData, err := json.MarshalIndent(cfgs, "", "    ")
	if err != nil {
		log.Printf("Error MarshalIndent config: %v", err)
		return err
	}

	err = os.WriteFile(cfgPath, jsonData, 0666)
	if err != nil {
		log.Printf("WriteFile error: %v", err)
		return err
	}

	return nil
}
