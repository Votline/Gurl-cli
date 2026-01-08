package core

import (
	"fmt"
	"gcli/internal/buffer"
	"gcli/internal/config"
	"gcli/internal/parser"
	"sync"

	"go.uber.org/zap"
)

func handleConfig(cPath, ckPath string) error {
	const op = "core.handleConfig"

	rb := buffer.NewRb()
	var wg sync.WaitGroup

	wg.Go(func() {
		for {
			cfg := rb.Read()
			if cfg == nil {
				return
			}
			fmt.Printf("%v", cfg)
		}
	})

	if err := parser.Parse(cPath, func(c config.Config) error {
		rb.Write(c)
		return nil
	}); err != nil {
		return fmt.Errorf("%s: %q: %w", op, cPath, err)
	}
	rb.Close()

	wg.Wait()
	return nil
}

func Start(cType, cPath, ckPath string, cCreate, ic bool, log *zap.Logger) error {
	if cCreate {
		return config.Create(cType, cPath)
	}
	return handleConfig(cPath, ckPath)
}
