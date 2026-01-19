package core

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"unsafe"

	"gcli/internal/buffer"
	"gcli/internal/config"
	"gcli/internal/parser"
	"gcli/internal/transport"

	"github.com/Votline/Gurlf"
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

	config.Init()
	sData, err := gurlf.ScanFile(cPath)
	if err != nil {
		return fmt.Errorf("%s: scan file %q: %w", op, cPath, err)
	}

	resHub := make([][]byte, 0, len(sData))
	cfgFileBuf := buffer.NewRb[config.Config]()
	rb := buffer.NewRb[config.Config]()
	resB := buffer.NewRb[*transport.Result]()
	transport.Init(resB.Write)

	var wg sync.WaitGroup
	wg.Go(func() {
		var err error
		for {
			cfg := rb.Read()
			if cfg == nil { break }
			cfgToFile := cfg.Clone()

			cfg.RangeDeps(func(d config.Dependency) {
				if d.TargetID >= len(resHub) {
				log.Error("Dependency points to non-exists config",
					zap.String("op", op),
					zap.Int("TargetID", d.TargetID))
					return
				}

				resp := resHub[d.TargetID]
				if resp == nil {
					log.Warn("Response for dependency is empty",
						zap.String("op", op),
						zap.Int("TargetID for resp", d.TargetID))
					return
				}

				inst := cfg.GetRaw(d.Key, d.Start, d.End)
				parser.ParseResponse(&resp, inst)
				cfg.Apply(d.Start, d.End, d.Key, resp)
			})

			res := resB.Read()

			execCfg := cfg.UnwrapExec()
			switch v := execCfg.(type) {
			case *config.HTTPConfig:
				err = transport.DoHTTP(v, res)
			case *config.GRPCConfig:
				cfg.Release()
				continue
			}

			fmt.Printf("\nEXEC:%T\nORIG:%T\n", execCfg, cfg)

			if err != nil {
				log.Error("Failed to send config",
					zap.String("op", op),
					zap.String("config name", cfg.GetName()),
					zap.String("config type", cfg.GetType()),
					zap.Error(err))
			}

			tmp := make([]byte, len(res.Raw))
			copy(tmp, res.Raw)
			resHub = append(resHub, tmp)

			resStr := unsafe.String(unsafe.SliceData(tmp), len(tmp))
			cfgToFile.SetResp(resStr)
			cfgFileBuf.Write(cfgToFile)

			cfg.Release()
			resB.Write(res)
		}
		cfgFileBuf.Close()
	})

	wg.Go(func(){
		f, err := os.OpenFile(cPath+".out", os.O_RDWR, 0644)
		if err != nil {
			log.Fatal("Failed to open file",
				zap.String("op", op),
				zap.String("path", cPath),
				zap.Error(err))
		}
		defer f.Close()

		/*st, err := f.Stat()
		if err != nil {
			log.Fatal("Failed to open file",
				zap.String("op", op),
				zap.String("path", cPath),
				zap.Error(err))
		}
		
		origSize := st.Size()
		curSize := origSize
		*/
		var buf bytes.Buffer
		cnt, bufSize := 0, 5
		
		for {
			cfg := cfgFileBuf.Read()
			if cfg == nil { break }

			data, err := gurlf.Marshal(cfg)
			if err != nil {
				log.Error("Failed to Marshal config, stop processing",
					zap.String("op", op),
					zap.Error(err))
			}

			buf.Write(data)
			cnt++

			if cnt == bufSize {
				cnt = 0
				if err := flush(&buf, f); err != nil {
					log.Error("failed to flush buffer",
						zap.String("op", op),
						zap.Error(err))
				}
			}

			cfg.Release()
		}

		if cnt > 0 {
			if err := flush(&buf, f); err != nil {
					log.Error("failed to flush buffer",
						zap.String("op", op),
						zap.Error(err))
				}
		}
	})

	if err := parser.ParseStream(&sData, rb.Write); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rb.Close()

	wg.Wait()
	return nil
}

func flush(buf *bytes.Buffer, f *os.File) error {
	const op = "core.flush"
	n, err := f.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("%s: write file: %w", op, err)
	}

	if n != buf.Len() {
		return fmt.Errorf("%s: short wrtie: expected %d, but got %d",
			op, buf.Len(), n)
	}

	buf.Reset()
	return nil
}
