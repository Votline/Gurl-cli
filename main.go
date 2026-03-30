package main

import (
	"fmt"
	"os"
	"slices"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Votline/Gurl-cli/internal/core"
)

const helpMsg = `
Usage: gcli <command> <args>

Commands:
	run <path>               Run config file
	create <path> <type>     Create config file
	help                     Show help
	args:
		-ic, --ignore-cert   Ignores site certificates
		-dp, --disable-print Disable printing response
		-d   --debug         Set debug log level
`

func parseArgs() (string, string, bool, bool, bool, bool, error) {
	const op = "main.parseArgs"

	var cfgType, cfgPath string
	var cfgCreate, ignoreCert, disPrint, debug bool

	args := os.Args[1:]
	command := args[0]

	switch command {
	case "run", "r", "--run", "-r":
		if len(args) < 2 {
			return "", "",
				false, false, false, false,
				fmt.Errorf("%s: Usage: gcli run <path> <args>", op)
		}
		cfgPath = args[1]

		ignoreCert = slices.Contains(args, "-ic") || slices.Contains(args, "--ignore-cert")
		disPrint = slices.Contains(args, "-dp") || slices.Contains(args, "--disable-print")
	case "create", "c", "--create", "-c":
		if len(args) <= 2 {
			return "", "",
				false, false, false, false,
				fmt.Errorf("%s: Usage: gcli create <path> <type>", op)
		}
		cfgPath = args[1]
		cfgType = args[2]
		cfgCreate = true
	case "help", "h", "--help", "-h":
		fmt.Print(helpMsg)
		return "", "", false, false, false, false, nil
	default:
		return "", "",
			false, false, false, false,
			fmt.Errorf("%s: Unknown command: %s", op, command)
	}

	debug = slices.Contains(args, "-d") || slices.Contains(args, "-dbg") || slices.Contains(args, "--debug")

	return cfgType, cfgPath, cfgCreate, ignoreCert, disPrint, debug, nil
}

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = ""
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.ConsoleSeparator = " | "
	lvl := zapcore.ErrorLevel

	cfgType, cfgPath, cfgCreate, ignoreCert, disPrint, debug, err := parseArgs()
	if err != nil {
		fmt.Println(err)
		return
	}
	if cfgPath == "" { // means "help" command
		return
	}

	if debug {
		lvl = zapcore.DebugLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	log, _ := cfg.Build()
	defer log.Sync()

	if err := core.Start(cfgType, cfgPath, cfgCreate, ignoreCert, disPrint, log); err != nil {
		log.Error("failed", zap.Error(err))
	}
}
