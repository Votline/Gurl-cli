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

func parseTypedConfig(rawCfg []byte) (Config, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawCfg, &head); err != nil {
		log.Printf("Error unmarshalling config: %v\nSource: %v",
			err, string(rawCfg))
		return nil, err
	}

	var c Config
	switch head.Type {
	case "http":
		c = &HTTPConfig{}
	case "grpc":
		c = &GRPCConfig{}
	case "repeated":
		c = &RepeatedConfig{}
	default:
		log.Printf("Invalid config type: %v", head.Type)
		return nil, errors.New("Invalid config type")
	}

	if err := json.Unmarshal(rawCfg, c); err != nil {
		log.Printf("Error unmarshalling config into result: %v\nSource: %v", err, string(rawCfg))
		return nil, errors.New("Unmarshling config error")
	}
	return c, nil
}

func Decode(cfgPath string) ([]Config, error) {
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
		configs := make([]Config, len(rawConfigs))
		for i, rawCfg := range rawConfigs {
			cfg, err := parseTypedConfig(rawCfg)
			if err != nil {
				log.Printf("ParseTypedConfig multi error: %v", err)
				return nil, fmt.Errorf("config: %d: %v", i, err)
			}
			configs[i] = cfg
		}
		return configs, nil
	}
	
	cfg, err := parseTypedConfig(data)
	if err != nil {
		log.Printf("ParseTypedConfig solo error: %v", err)
		return nil, err
	}
	return []Config{cfg}, nil
}

func ConfigUpd[T Config](parsed T, cfgPath string) error {
	cfgs, err := Decode(cfgPath)
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
