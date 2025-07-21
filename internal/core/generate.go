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
	if cfgType == "http" {
		cfg = config.SetupHTTP()
	} else {
		cfg = config.SetupGRPC()
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
