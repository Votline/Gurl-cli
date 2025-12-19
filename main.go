package main

import (
	"flag"
	"fmt"
	"os"

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

	var cfgCreate bool
	flag.BoolVar(&cfgCreate, "config-create", false, "Creates a configuration file (.json). Default for HTTP requests")

	var ignoreCert bool
	flag.BoolVar(&ignoreCert, "ignore-cert", false, "Ignores site certificates (https)")
	flag.BoolVar(&ignoreCert, "ic", false, "Ignores site certificates (https)")

	var cookiePath string
	flag.StringVar(&cookiePath, "cookie", "", "cookie.txt path (it can be used for the following configuration. In-memory is used for the current configuration.)")

	logLevel := flag.String("level", "info", "Log level (debug, info, warn, error, dpanic, panic, fatal)")

	cfgType := flag.String("type", "http", "Sets the request type in the configuration file(type field in .json")

	defPath, _ := os.Getwd()
	cfgPath := flag.String("config", defPath, "Specifies the name and path for creating the configuration file")

	flag.Parse()

	parsedLevel := zapcore.InfoLevel
	if lvl, err := zapcore.ParseLevel(*logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level %q, defaulting to info\n", *logLevel)
	} else {
		parsedLevel = lvl
	}
	cfg.Level = zap.NewAtomicLevelAt(parsedLevel)
	
	log, _ := cfg.Build()
	defer log.Sync()

	core.Start(*cfgType, *cfgPath, cfgCreate, ignoreCert, cookiePath, log)
}
