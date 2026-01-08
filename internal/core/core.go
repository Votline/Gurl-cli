package core

import (
	"fmt"
	"gcli/internal/config"
	"gcli/internal/parser"
	"gcli/internal/buffer"

	"go.uber.org/zap"
)

func handleConfig(cPath, ckPath string) error {
	const op = "core.handleConfig"

	rb := buffer.NewRb()
	if err := parser.Parse(cPath, func(c config.Config) error {
		rb.Write(c)
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
