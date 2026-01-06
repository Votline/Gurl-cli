package core

import (
	"fmt"
	"gcli/internal/config"
	"gcli/internal/parser"

	"go.uber.org/zap"
)

func handleConfig(cPath, ckPath string) error {
	const op = "core.handleConfig"

	cfgs, err := parser.Parse(cPath)
	if err != nil {
		return fmt.Errorf("%s: %q: %w", op, cPath, err)
	}

	for i, cfg := range cfgs {
		if cfg == nil {
			return fmt.Errorf("%s: cfg №[%d]: is nil", op, i)
		}

		fmt.Printf("%v", cfg)

		switch v := cfg.(type) {
		case *config.HTTPConfig:
			continue
		case *config.GRPCConfig:
			continue
		default:
			return fmt.Errorf("%s: cfg №[%d]: undefined type: %T",
				op, i, v)
		}
	}

	return nil
}

func Start(cType, cPath, ckPath string, cCreate, ic bool, log *zap.Logger) error {
	if cCreate {
		return config.Create(cType, cPath)
	}
	return handleConfig(cPath, ckPath)
}
