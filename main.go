package main

import (
	"os"
	"flag"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"Gurl-cli/internal/core"
)

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = ""
	cfg.EncoderConfig.EncodeLevel = func(l zapcore.Level, pae zapcore.PrimitiveArrayEncoder) {
		pae.AppendString("\n" + l.CapitalString())
	}
	log, _ := cfg.Build()
	defer log.Sync()

	var cfgCreate bool
	flag.BoolVar(&cfgCreate, "config-create", false, "Creates a configuration file (.json). Default for HTTP requests")

	var ignoreCert bool
	flag.BoolVar(&ignoreCert, "ignore-cert", false, "Ignores site certificates (https)")
	flag.BoolVar(&ignoreCert, "ic", false, "Ignores site certificates (https)")

	var cookiePath string
	flag.StringVar(&cookiePath, "cookie", "", "cookie.txt path (it can be used for the following configuration. In-memory is used for the current configuration.)")

	cfgType := flag.String("type", "http", "Sets the request type in the configuration file(type field in .json")

	defPath, _ := os.Getwd()
	cfgPath := flag.String("config", defPath, "Specifies the name and path for creating the configuration file")

	flag.Parse()

	core.Start(*cfgType, *cfgPath, cfgCreate, ignoreCert, cookiePath, log)
}
