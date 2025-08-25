package generate

import (
	"os"
	"log"
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
	if err != nil {
		log.Printf("Couldn't marshal config: %v", err)
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
		log.Printf("Couldn't create a directory: %v", err)
		return err
	}

	if err := os.WriteFile(path, json, 0644); err != nil {
		log.Printf("Couldn't create a file: %v", err)
		return err
	}

	return nil
}
