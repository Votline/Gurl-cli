package generate

import (
	"os"
	"path/filepath"
	"encoding/json"
	
	"go.uber.org/zap"

	"Gurl-cli/internal/config"
)

func InitConfig(path, cfgType string, log *zap.Logger) error {
	var cfg any
	switch cfgType {
	case "grpc":
		cfg = config.SetupGRPC()
	case "http":
		cfg = config.SetupHTTP()
	default:
		cfg = config.SetupRepeated()
	}

	json, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		log.Error("Couldn't marshal config", zap.Error(err))
		return err
	}

	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		path = filepath.Join(path, cfgType + "_config.json")
	} else {
		if filepath.Ext(path) != ".json" {
			path += ".json"
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Error("Couldn't create a directory", zap.Error(err))
		return err
	}

	if err := os.WriteFile(path, json, 0644); err != nil {
		log.Error("Couldn't create a file", zap.Error(err))
		return err
	}

	return nil
}
