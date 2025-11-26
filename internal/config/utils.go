package config

import (
	"os"
	"errors"
	"encoding/json"
	"path/filepath"

	"go.uber.org/zap"
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

func (p *Parser) parseTypedConfig(rawCfg []byte) (Config, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawCfg, &head); err != nil {
		p.log.Error("Unmarshalling config error",
			zap.String("source", string(rawCfg)),
			zap.Error(err))
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
		p.log.Warn("Invalid config type", zap.String("type", head.Type))
		return nil, errors.New("Invalid config type")
	}

	if err := json.Unmarshal(rawCfg, c); err != nil {
		p.log.Error("Unmarshalling config error",
			zap.String("source", string(rawCfg)),
			zap.Error(err))
		return nil, errors.New("Unmarshling config error")
	}
	return c, nil
}

func (p *Parser) Decode(cfgPath string) ([]Config, error) {
	path, err := findConfigPath(cfgPath)
	if err != nil {
		p.log.Error("Couldn't find config path error", zap.Error(err))
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		p.log.Error("ReadFile error", zap.Error(err))
		return nil, err
	}
	
	var rawConfigs []json.RawMessage
	if err := json.Unmarshal(data, &rawConfigs); err == nil {
		configs := make([]Config, len(rawConfigs))
		for i, rawCfg := range rawConfigs {
			cfg, err := p.parseTypedConfig(rawCfg)
			if err != nil {
				p.log.Error("ParseTypedConfig multi error", zap.Error(err))

				return nil, err
			}
			configs[i] = cfg
		}
		return configs, nil
	}
	
	cfg, err := p.parseTypedConfig(data)
	if err != nil {
		p.log.Error("Parse one config error", zap.Error(err))
		return nil, err
	}
	return []Config{cfg}, nil
}

func (p *Parser) ConfigUpd(parsed Config, cfgPath string) error {
	cfgs, err := p.Decode(cfgPath)
	if err != nil {
		p.log.Error("Decode config error",zap.Error(err))
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
		p.log.Error("Error MarshalIndent config", zap.Error(err))
		return err
	}

	err = os.WriteFile(cfgPath, jsonData, 0666)
	if err != nil {
		p.log.Error("WriteFile error", zap.Error(err))
		return err
	}

	return nil
}
