package main

import (
	"os"
	"flag"

	"go.uber.org/zap"

	"Gurl-cli/internal/core"
)



func main() {
	log, _ := zap.NewDevelopment()

	var cfgCreate bool
	flag.BoolVar(&cfgCreate, "config-create", false, "Creates a configuration file (.json). Default for HTTP requests")

	var ignoreCert bool
	flag.BoolVar(&ignoreCert, "ignore-cert", false, "Ignores site certificates (https)")
	flag.BoolVar(&ignoreCert, "ic", false, "Ignores site certificates (https)")

	cfgType := flag.String("type", "http", "Sets the request type in the configuration file(type field in .json")

	defPath, _ := os.Getwd()
	cfgPath := flag.String("config", defPath, "Specifies the name and path for creating the configuration file")

	flag.Parse()

	core.Start(*cfgType, *cfgPath, cfgCreate, ignoreCert, log)
}
