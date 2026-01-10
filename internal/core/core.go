package core

import (
	"fmt"
	"sync"

	"gcli/internal/buffer"
	"gcli/internal/config"
	"gcli/internal/parser"
	"gcli/internal/transport"

	"go.uber.org/zap"
)

func Start(cType, cPath, ckPath string, cCreate, ic bool, log *zap.Logger) error {
	if cCreate {
		return config.Create(cType, cPath)
	}
	return handleConfig(cPath, ckPath, log)
}

func handleConfig(cPath, ckPath string, log *zap.Logger) error {
	const op = "core.handleConfig"

	rb := buffer.NewRb[config.Config]()
	resB := buffer.NewRb[*transport.Result]()
	transport.Init(resB.Write)
	var wg sync.WaitGroup

	wg.Go(func() {
		var err error
		for {
			cfg := rb.Read()
			if cfg == nil {
				return
			}

			res := resB.Read()
			switch v := cfg.(type) {
			case *config.HTTPConfig:
				err = transport.DoHTTP(v, res)
			case *config.GRPCConfig:
				cfg.Release()
				continue
			}

			if err != nil {
				log.Error("Failed to send config",
					zap.String("op", op),
					zap.String("config name", cfg.GetName()),
					zap.String("config type", cfg.GetType()),
					zap.Error(err))
			}
			cfg.Release()

			fmt.Printf("res: %v\n", string(res.Raw))

			resB.Write(res)
		}
	})

	if err := parser.Parse(cPath, rb.Write); err != nil {
		return fmt.Errorf("%s: %q: %w", op, cPath, err)
	}
	rb.Close()

	wg.Wait()
	return nil
}
