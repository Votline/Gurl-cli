package core

import (
	"fmt"
	"sync"

	"gcli/internal/buffer"
	"gcli/internal/config"
	"gcli/internal/parser"
	"gcli/internal/transport"

	"go.uber.org/zap"
	"github.com/Votline/Gurlf"
)

func Start(cType, cPath, ckPath string, cCreate, ic bool, log *zap.Logger) error {
	if cCreate {
		return config.Create(cType, cPath)
	}
	return handleConfig(cPath, ckPath, log)
}

func handleConfig(cPath, ckPath string, log *zap.Logger) error {
	const op = "core.handleConfig"

	config.Init()
	sData, err := gurlf.ScanFile(cPath)
	if err != nil {
		return fmt.Errorf("%s: scan file %q: %w", op, cPath, err)
	}

	hub := make([][]byte, 0, len(sData))
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

			cfg.RangeDeps(func(d config.Dependency) {
				if d.TargetID >= len(hub) {
				log.Error("Dependency points to non-exists config",
					zap.String("op", op),
					zap.Int("TargetID", d.TargetID))
					return
				}

				resp := hub[d.TargetID]
				if resp == nil {
					log.Warn("Response for dependency is empty",
						zap.String("op", op),
						zap.Int("TargetID for resp", d.TargetID))
					return
				}

				cfg.Apply(d.Start, d.End, d.Key, resp)
			})

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
			tmp := make([]byte, len(res.Raw))
			copy(tmp, res.Raw)
			hub = append(hub, tmp)

			resB.Write(res)
		}
	})

	if err := parser.ParseStream(&sData, rb.Write); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rb.Close()

	wg.Wait()
	return nil
}
