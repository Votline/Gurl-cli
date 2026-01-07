package core

import (
	"fmt"
	"gcli/internal/config"
	"gcli/internal/parser"

	"go.uber.org/zap"
)

func handleConfig(cPath, ckPath string) error {
	const op = "core.handleConfig"

	if err := parser.Parse(cPath, func(c config.Config) error {
		fmt.Printf("%v", c)
		return nil
	}); err != nil {
		return fmt.Errorf("%s: %q: %w", op, cPath, err)
	}

	return nil
}

func Start(cType, cPath, ckPath string, cCreate, ic bool, log *zap.Logger) error {
	if cCreate {
		return config.Create(cType, cPath)
	}
	return handleConfig(cPath, ckPath)
}
