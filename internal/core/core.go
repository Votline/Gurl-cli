package core

import (
	"bytes"
	"fmt"
	"io"
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

type DepBindigs struct {
	From func(res *transport.Result) []byte
	To   func(cfg config.Config, start, end int, key string, val []byte)
}

var depBindings = map[string]DepBindigs{
	"RESPONSE": {
		From: func(res *transport.Result) []byte { return res.Raw },
		To: func(cfg config.Config, s, e int, k string, v []byte) {
			raw := cfg.GetRaw(k, s, e)
			parser.ParseResponse(&v, raw)
			cfg.Apply(s, e, k, v)
		},
	},
	"COOKIES": {
		From: func(res *transport.Result) []byte { return res.Cookie },
		To: func(cfg config.Config, s, e int, k string, v []byte) {
			cfg.Apply(s, e, k, v)
		},
	},
}

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

	resHub := make([]*transport.Result, 0, len(sData))
	cfgFileBuf := buffer.NewRb[config.Config]()
	rb := buffer.NewRb[config.Config]()
	resB := buffer.NewRb[*transport.Result]()
	trnsp := transport.NewTransport(resB.Write)

	var wg sync.WaitGroup
	wg.Go(func() {
		var err error
		for {
			cfg := rb.Read()
			if cfg == nil {
				break
			}

			cfgToFile := cfg.Clone()

			cfg.RangeDeps(func(d config.Dependency) {
				bind, ok := depBindings[d.InsTp]
				if !ok {
					log.Error("Dependency points to non-exists key",
						zap.String("op", op),
						zap.String("key", d.InsTp))
					return
				}

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

				val := bind.From(resp)

				log.Warn("COOKIES val",
					zap.Int("len", len(val)),
					zap.ByteString("val", val),
				)

				bind.To(cfg, d.Start, d.End, d.Key, val)
			})

			res := resB.Read()

			execCfg := cfg.UnwrapExec()
			switch v := execCfg.(type) {
			case *config.HTTPConfig:
				err = trnsp.DoHTTP(v, res)
			case *config.GRPCConfig:
				cfg.Release()
				cfgToFile.ReleaseClone()
				continue
			}

			if err != nil {
				log.Error("Failed to send config",
					zap.String("op", op),
					zap.String("config name", cfg.GetName()),
					zap.String("config type", cfg.GetType()),
					zap.Error(err))
			}
			resHub = append(resHub, res)

			tmp := make([]byte, len(res.Raw))
			copy(tmp, res.Raw)

			resStr := unsafe.String(unsafe.SliceData(tmp), len(tmp))
			cfgToFile.SetResp(resStr)
			cfgToFile.SetCookie(res.Cookie)

			cfgFileBuf.Write(cfgToFile)

			cfg.Release()

			resB.Write(res)
		}
		cfgFileBuf.Close()
	})

	wg.Go(func() {
		cnt, bufSize := 0, 5
		var buf bytes.Buffer
		var pendingOffset int64
		var cfg config.Config

		tmpPath := cPath + ".out.tmp"
		f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			log.Fatal("Failed to open file",
				zap.String("op", op),
				zap.String("path", cPath),
				zap.Error(err))
		}
		defer func() {
			if r := recover(); r != nil {
				log.Error("Recovered from panic",
					zap.String("op", op),
					zap.Any("recovered", r))

				if err := flush(&buf, f); err != nil {
					log.Error("failed to flush buffer",
						zap.String("op", op),
						zap.Error(err))
				}

				orig, err := os.Open(cPath)
				if err == nil {
					if _, err = orig.Seek(pendingOffset, io.SeekStart); err == nil {
						if _, err := io.Copy(f, orig); err != nil {
							log.Error("Failed to copy file",
								zap.String("op", op),
								zap.String("path", cPath),
								zap.Error(err))
						}
					} else {
						log.Error("Failed to seek file",
							zap.String("op", op),
							zap.String("path", cPath),
							zap.Error(err))
					}
				} else {
					log.Error("Failed to open file",
						zap.String("op", op),
						zap.String("path", cPath),
						zap.Error(err))
				}
				defer orig.Close()
			}

			if err := f.Sync(); err != nil {
				log.Error("Failed to sync file",
					zap.String("op", op),
					zap.String("path", cPath),
					zap.Error(err))
			}

			f.Close()
			if err := os.Rename(tmpPath, cPath); err != nil {
				log.Error("Failed to rename file",
					zap.String("op", op),
					zap.String("path to", cPath),
					zap.String("path from", tmpPath),
					zap.Error(err))
			}
		}()

		for {
			cfg = cfgFileBuf.Read()
			if cfg == nil {
				break
			}

			data, err := gurlf.Marshal(cfg)
			if err != nil {
				log.Error("Failed to Marshal config",
					zap.String("op", op),
					zap.Error(err))
				continue
			}
			cfg.ReleaseClone()

			buf.Write(data)
			cnt++
			pendingOffset = int64(cfg.GetEnd())

			if cnt == bufSize {
				cnt = 0
				if err := flush(&buf, f); err != nil {
					log.Error("failed to flush buffer",
						zap.String("op", op),
						zap.Error(err))
				}
			}
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

	if buf.Len() == 0 {
		return nil
	}

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
