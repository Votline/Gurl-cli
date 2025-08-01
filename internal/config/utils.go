package config

import (
	"os"
	"fmt"
	"errors"
	"reflect"
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

func parseTypedConfig(rawCfg []byte) (interface{}, error) {
	var cfgType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawCfg, &cfgType); err != nil {
		return nil, err
	}

	var cfg interface{}
	switch cfgType.Type {
	case "http":
		cfg = &HTTPConfig{}
	case "grpc":
		cfg = &GRPCConfig{}
	default:
		return nil, errors.New("Unkown config type: "+cfgType.Type)
	}
	if err := json.Unmarshal(rawCfg, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Decode(cfgPath string) (interface{}, error) {
	path, err := findConfigPath(cfgPath)
	if err != nil {return nil, err}

	data, err := os.ReadFile(path)
	if err != nil {return nil, err}
	
	var rawConfigs []json.RawMessage
	if err := json.Unmarshal(data, &rawConfigs); err == nil {
		configs := make([]interface{}, len(rawConfigs))
		for i, rawCfg := range rawConfigs {
			cfg, err := parseTypedConfig(rawCfg)

			if err != nil {
				return nil, fmt.Errorf("config: %d: %v", i, cfg)
			}
			configs[i] = cfg
		}
		return configs, nil
	}
	return parseTypedConfig(data)
}

func ConfigUpd(cfg interface{}, cfgPath string) error {
	cfgs, err := Decode(cfgPath)
	if err != nil {return err}

	cfgVal := reflect.ValueOf(cfg)
	if cfgVal.Kind() == reflect.Ptr {
		cfgVal = cfgVal.Elem()
	}
	cfgID := cfgVal.FieldByName("ID").String()
	if v, ok := cfgs.([]interface{}); ok {
		for i, c := range v {
			cVal := reflect.ValueOf(c)
			if cVal.Kind() == reflect.Ptr {
				cVal = cVal.Elem()
			}
			cID := cVal.FieldByName("ID").String()
			if cID == cfgID {
				v[i] = cfg
				cfgs = v
				break
			}
		}
	} else {
		cfgs = cfg
	}

	jsonData, err := json.MarshalIndent(cfgs, "", "    ")
	if err != nil {return err}

	err = os.WriteFile(cfgPath, jsonData, 0666)
	if err != nil {return err}

	return nil
}
