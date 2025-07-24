package main

import (
	"flag"
	"os"

	"Gurl-cli/internal/core"
)

func main() {
	var cfgCreate bool
	flag.BoolVar(&cfgCreate, "config-create", false, "Creates a configuration file (.json). Default for HTTP requests")

	cfgType := flag.String("type", "http", "Sets the request type in the configuration file(type field in .json")

	defPath, _ := os.Getwd()
	cfgPath := flag.String("config", defPath, "Specifies the name and path for creating the configuration file")
	flag.Parse()

	core.HandleFlags(*cfgType, *cfgPath, cfgCreate)
}
