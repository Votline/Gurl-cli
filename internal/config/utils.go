package config

import (
	"os"
	"fmt"
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
	var cfg T
	if err := json.Unmarshal(rawCfg, &cfg); err != nil {
		var zero T
		return zero, err
	}

	return cfg, nil
}

func Decode[T Config](cfgPath string) ([]T, error) {
	path, err := findConfigPath(cfgPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var rawConfigs []json.RawMessage
	if err := json.Unmarshal(data, &rawConfigs); err == nil {
		configs := make([]T, len(rawConfigs))
		for i, rawCfg := range rawConfigs {
			cfg, err := parseTypedConfig[T](rawCfg)
			if err != nil {
				return nil, fmt.Errorf("config: %d: %v", i, err)
			}
			configs[i] = cfg
		}
		return configs, nil
	}
	
	cfg, err := parseTypedConfig[T](data)
	if err != nil {
		return nil, err
	}
	return []T{cfg}, nil
}

func ConfigUpd[T Config](parsed T, cfgPath string) error {
	cfgs, err := Decode[T](cfgPath)
	if err != nil {
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
		return err
	}

	err = os.WriteFile(cfgPath, jsonData, 0666)
	if err != nil {
		return err
	}

	return nil
}
