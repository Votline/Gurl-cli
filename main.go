package main

import (
	"flag"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gcli/internal/core"
)

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = ""
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.ConsoleSeparator = " | "
	lvl := zapcore.ErrorLevel

	cfgCreate := flag.Bool("config-create", false, "Creates config by type. Defaul HTTP")

	ignoreCert := flag.Bool("ignore-cert", false, "Ignores site certificates")
	ignoreCert = flag.Bool("ic", false, "Ignores site certificates")

	ckPath := flag.String("cookie", "", "cookie.txt path (it can be used for the following configuration. In-memory is used for the current configuration.)")

	cfgType := flag.String("type", "http", "Sets the request type in the configuration file(type field in .json")

	defPath, _ := os.Getwd()
	cfgPath := flag.String("config", defPath, "Specifies the name and path for creating the config")

	dbg := flag.Bool("debug", false, "Set debug log level")

	flag.Parse()

	if *dbg {
		lvl = zapcore.DebugLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	log, _ := cfg.Build()
	defer log.Sync()

	if err := core.Start(*cfgType, *cfgPath, *ckPath, *cfgCreate, *ignoreCert, log); err != nil {
		log.Error("failed", zap.Error(err))
	}
}
