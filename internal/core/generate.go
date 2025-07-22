package core

import (
	"os"
	"fmt"
	"path/filepath"
	"encoding/json"

	"Gurl-cli/internal/config"
)

func InitConfig(path, cfgType string) error {
	var cfg interface{}
	if cfgType == "grpc" {
		cfg = config.SetupGRPC()
	} else if cfgType == "mixed" {
		cfg = config.SetupMixed()
	} else {
		cfg = config.SetupHTTP()
	}

	json, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {return err}

	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		path = fmt.Sprintf("%s/%s_%s", path, cfgType, "config.json")
	} else {
		if filepath.Ext(path) != ".json" {
			path += ".json"
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, json, 0644); err != nil {
		return err
	}

	return nil
}
