package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"gcli/internal/buffer"
	"gcli/internal/config"
	"gcli/internal/parser"
	"gcli/internal/transport"

	"github.com/Votline/Gurlf"
	"go.uber.org/zap"
)

const importConfigCode = -1000

type DepBindigs struct {
	From func(res *transport.Result) []byte
	To   func(cfg config.Config, start, end int, key string, val, inst []byte)
}

var depBindings = map[string]DepBindigs{
	"RESPONSE": {
		From: func(res *transport.Result) []byte { return res.Raw },
		To: func(cfg config.Config, s, e int, k string, v, inst []byte) {
			parser.ParseResponse(&v, inst)
			cfg.Apply(s, e, k, v)
		},
	},
	"COOKIES": {
		From: func(res *transport.Result) []byte { return res.Cookie },
		To: func(cfg config.Config, s, e int, k string, v, inst []byte) {
			cfg.Apply(s, e, k, v)
			cfg.SetFlag(config.FlagUseFileCookies)
		},
	},
}

func Start(cType, cPath string, cCreate, ic, disablePrint bool, log *zap.Logger) error {
	if cCreate {
		return config.Create(cType, cPath)
	}
	config.Init()
	return handleConfig(cPath, disablePrint, log)
}

func handleConfig(cPath string, disablePrint bool, log *zap.Logger) error {
	const op = "core.handleConfig"

	sData, err := gurlf.ScanFile(cPath)
	if err != nil {
		return fmt.Errorf("%s: scan file %q: %w", op, cPath, err)
	}

	resHub := make([]*transport.Result, 0, len(sData))
	cfgFileBuf := buffer.NewRb[config.Config]()
	rb := buffer.NewRb[config.Config]()
	resB := buffer.NewRb[*transport.Result]()
	resPrintBuf := buffer.NewRb[*transport.Result]()
	trnsp := transport.NewTransport(resB.Write)

	if disablePrint {
		resPrintBuf = buffer.NewNop[*transport.Result]()
	}

	isCrashed := false
	var globalErr error
	var wg sync.WaitGroup
	wg.Go(func() {
		defer cfgFileBuf.Close()
		defer resPrintBuf.Close()

		for {
			cfg := rb.Read()
			if cfg == nil {
				break
			}

			for {
				cfgToFile := cfg.Clone()
				log.Debug("processing config",
					zap.String("op", op),
					zap.String("name", cfg.GetName()),
					zap.Int("id", cfg.GetID()))

				applyDeps(cfg, &resHub, log)

				res := resB.Read()
				execCfg := cfg.UnwrapExec()

				applyWait(cfg, execCfg, log)

				if impCfg, ok := cfg.(*config.ImportConfig); ok {
					log.Debug("import config",
						zap.String("op", op),
						zap.String("name", cfg.GetName()),
						zap.Int("id", cfg.GetID()))
					handleConfig(impCfg.TargetPath, disablePrint, log)
					res.Info.Code = importConfigCode
				} else {
					sendConfig(cfg, execCfg, trnsp, res, log)
				}

				res.CfgID = cfg.GetID()

				resHub = append(resHub, res)

				resPrintBuf.Write(res)

				id := applyExpect(cfg, execCfg, res, log)

				if !isCrashed {
					cfgToFile.Update(res.Raw, res.Cookie)
					cfgFileBuf.Write(cfgToFile)
				}

				if id == parser.ExpectDone && !isCrashed {
					break
				} else if id == parser.ExpectDone && isCrashed {
					cfg.Release()
					resB.Write(res)
					return
				}

				if id == parser.ExpectCrash {
					cfg.Release()
					resB.Write(res)
					return
				}

				if isCrashed {
					cfg.Release()
					resB.Write(res)
					return
				}

				isCrashed = true
				var nextCfg config.Config
				origEnd := cfg.GetEnd()

				if err := parser.ParseFindConfig(&sData, &nextCfg, id); err != nil {
					log.Error("Failed to find config",
						zap.String("op", op),
						zap.String("name", cfg.GetName()),
						zap.Int("id", cfg.GetID()),
						zap.Int("target", id),
						zap.Error(err))
					cfg.Release()
					resB.Write(res)
					break
				}

				cfg.Release()
				resB.Write(res)
				cfg = nextCfg
				cfg.SetEnd(origEnd)
			}
		}
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
				copyTail(f, cPath, &buf, pendingOffset, log)
			} else if isCrashed {
				log.Debug("Copy tail after jump",
					zap.String("op", op),
					zap.String("path", cPath))
				copyTail(f, cPath, &buf, pendingOffset, log)
			}

			flush(&buf, f)

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
				return
			}

			data, err := gurlf.Marshal(cfg)
			if err != nil {
				cfg.ReleaseClone()
				log.Error("Failed to Marshal config",
					zap.String("op", op),
					zap.Error(err))
				continue
			}

			log.Debug("marshaled config",
				zap.String("op", op),
				zap.String("name", cfg.GetName()))

			buf.Write(data)
			cnt++
			pendingOffset = int64(cfg.GetEnd())

			log.Debug("wrote config",
				zap.String("op", op),
				zap.String("name", cfg.GetName()),
				zap.Int("size", cnt),
				zap.Int64("pendingOffset", pendingOffset))

			cfg.ReleaseClone()

			if cnt == bufSize {
				cnt = 0
				if err := flush(&buf, f); err != nil {
					log.Error("failed to flush buffer",
						zap.String("op", op),
						zap.Error(err))
				}

				log.Debug("flushed buffer",
					zap.String("op", op),
					zap.String("name", cfg.GetName()))

			}
		}
	})

	if !disablePrint {
		wg.Go(func() {
			for {
				res := resPrintBuf.Read()
				if res == nil {
					break
				}

				if err := prettyPrint(res); err != nil {
					log.Error("Failed to print response",
						zap.String("op", op),
						zap.Error(err))
				}
			}
		})
	}

	if err := parser.ParseStream(&sData, rb.Write, log); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rb.Close()

	wg.Wait()

	if globalErr != nil {
		return fmt.Errorf("%s: %w", op, globalErr)
	}
	return nil
}

func applyDeps(cfg config.Config, resHub *[]*transport.Result, log *zap.Logger) {
	const op = "core.applyDeps"

	allDeps := make([]config.Dependency, 0, cfg.GetDepsLen())
	cfg.RangeDeps(func(d config.Dependency) {
		allDeps = append(allDeps, d)
	})
	sort.Slice(allDeps, func(i, j int) bool {
		return allDeps[i].Start > allDeps[j].Start
	})

	for _, d := range allDeps {
		if d.Key == "Response" {
			continue
		}

		if d.TargetID == config.DataFromFile {
			cfg.SetFlag(config.FlagUseFileCookies)
			cfg.Apply(d.Start, d.End, d.Key, nil)
			continue
		} else if d.TargetID == config.RandomData {
			rawSnapshot := make([]byte, len(cfg.GetRaw(d.Key)))
			copy(rawSnapshot, cfg.GetRaw(d.Key))
			if d.End > len(rawSnapshot) || d.End > cap(rawSnapshot) {
				log.Error("Invalid random data",
					zap.String("op", op),
					zap.String("key", d.Key),
					zap.Int("start", d.Start),
					zap.Int("end", d.End))
				continue
			}
			instructionBytes := rawSnapshot[d.Start:d.End]

			val := make([]byte, 36)
			parser.ParseRandom(instructionBytes, &val)
			if val == nil {
				log.Error("Failed to parse random",
					zap.String("op", op),
					zap.String("key", d.Key),
					zap.String("inst", string(instructionBytes)))
				continue
			}

			cfg.Apply(d.Start, d.End, d.Key, val)

			continue
		}

		bind, ok := depBindings[d.InsTp]
		if !ok {
			log.Error("Dependency points to non-exists key",
				zap.String("op", op),
				zap.String("key", d.InsTp))
			continue
		}

		if d.TargetID >= len(*resHub) {
			log.Error("Dependency points to non-exists config",
				zap.String("op", op),
				zap.Int("TargetID", d.TargetID))
			continue
		}

		resp := (*resHub)[d.TargetID]
		if resp == nil {
			log.Warn("Response for dependency is empty",
				zap.String("op", op),
				zap.Int("TargetID for resp", d.TargetID))
			continue
		}

		log.Debug("Dependency",
			zap.String("op", op),
			zap.Int("TargetID for resp", d.TargetID),
			zap.String("key", d.Key),
			zap.String("name", cfg.GetName()))

		val := bind.From(resp)

		rawSnapshot := make([]byte, len(cfg.GetRaw(d.Key)))
		copy(rawSnapshot, cfg.GetRaw(d.Key))

		instructionBytes := rawSnapshot[d.Start:d.End]

		bind.To(cfg, d.Start, d.End, d.Key, val, instructionBytes)

		log.Debug("applied dependencies",
			zap.String("op", op),
			zap.String("name", cfg.GetName()),
			zap.String("key", d.Key),
			zap.String("inst", string(instructionBytes)),
			zap.String("val", string(val)))
	}
}

func applyWait(cfg config.Config, execCfg config.Config, log *zap.Logger) {
	const op = "core.applyWait"

	if cfg.GetWait() == nil && execCfg.GetWait() != nil {
		cfg.SetWait(execCfg.GetWait())
	}

	dur := parser.ParseWait(cfg.GetWait())
	if dur == parser.Error {
		waitStr := unsafe.String(unsafe.SliceData(cfg.GetWait()), len(cfg.GetWait()))
		log.Error("Failed to parse wait",
			zap.String("op", op),
			zap.String("name", cfg.GetName()),
			zap.String("wait", waitStr))
	} else if dur != 0 {
		log.Debug("sleep",
			zap.String("op", op),
			zap.String("name", cfg.GetName()),
			zap.Duration("dur", dur),
			zap.String("wait", string(cfg.GetWait())))
		time.Sleep(dur)
	}
}

func sendConfig(cfg config.Config, execCfg config.Config, trnsp *transport.Transport, res *transport.Result, log *zap.Logger) {
	const op = "core.sendConfig"

	var err error
	switch v := execCfg.(type) {
	case *config.HTTPConfig:
		err = trnsp.DoHTTP(v, res)
		log.Debug("send http",
			zap.String("op", op),
			zap.String("url", string(v.URL)),
			zap.String("method", string(v.Method)),
			zap.String("body", string(v.Body)),
			zap.String("headers", string(v.Headers)),
			zap.String("cookie", string(v.CookieIn)))

	case *config.GRPCConfig:
		log.Debug("send grpc",
			zap.String("op", op),
			zap.String("target", string(v.Target)),
			zap.String("endpoint", string(v.Endpoint)),
			zap.String("body", string(v.Data)),
			zap.String("protoPath", string(v.ProtoPath)),
			zap.String("importPaths", string(v.ImportPaths)),
			zap.String("dialOpts", string(v.DialOpts)))

		err = transport.DoGRPC(v, res)
	}

	if err != nil {
		log.Error("Failed to send config",
			zap.String("op", op),
			zap.String("config name", cfg.GetName()),
			zap.String("config type", cfg.GetType()),
			zap.Error(err))
	}
}

func applyExpect(cfg config.Config, execCfg config.Config, res *transport.Result, log *zap.Logger) int {
	const op = "core.applyExpect"

	if cfg.GetExpect() == nil && execCfg.GetExpect() != nil {
		cfg.SetExpect(execCfg.GetExpect())
	}

	if id := parser.ParseExpect(cfg.GetExpect(), res.Info.Code); id == parser.Error {
		expStr := unsafe.String(unsafe.SliceData(cfg.GetExpect()), len(cfg.GetExpect()))
		log.Error("Failed to parse expect",
			zap.String("op", op),
			zap.String("config name", cfg.GetName()),
			zap.Int("config id", cfg.GetID()),
			zap.String("expected", expStr))
		return parser.ExpectDone
	} else if id != parser.ExpectDone {
		expStr := unsafe.String(unsafe.SliceData(cfg.GetExpect()), len(cfg.GetExpect()))
		log.Error("Expected fail",
			zap.String("op", op),
			zap.String("config name", cfg.GetName()),
			zap.Int("config id", cfg.GetID()),
			zap.Int("response code", res.Info.Code),
			zap.String("expected", expStr),
			zap.Int("expected id", id))

		if id == parser.ExpectCrash {
			log.Debug("Expected action",
				zap.String("op", op),
				zap.String("action", "crash"))
			return parser.ExpectCrash
		} else if id < 0 {
			log.Debug("Expected action",
				zap.String("op", op),
				zap.String("action", "ignore"))
			return parser.ExpectDone
		}

		log.Debug("Expected action",
			zap.String("op", op),
			zap.Int("action: goto to id", id))

		return id
	}
	return parser.ExpectDone
}

func copyTail(f *os.File, cPath string, buf *bytes.Buffer, pendingOffset int64, log *zap.Logger) {
	const op = "core.copyTail"

	if err := flush(buf, f); err != nil {
		log.Error("failed to flush buffer", zap.String("op", op), zap.Error(err))
	}

	orig, err := os.Open(cPath)
	if err != nil {
		log.Error("Failed to open file", zap.String("op", op), zap.Error(err))
		return
	}
	defer orig.Close()

	if _, err = orig.Seek(pendingOffset, io.SeekStart); err != nil {
		log.Error("Failed to seek file", zap.String("op", op), zap.Error(err))
		return
	}

	if _, err := io.Copy(f, orig); err != nil {
		log.Error("Failed to copy tail", zap.String("op", op), zap.Error(err))
	}
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

func prettyPrint(res *transport.Result) error {
	const op = "core.prettyPrint"

	if res.Info.Code == importConfigCode {
		return nil
	}

	fmt.Println(strings.Repeat("-", 20))

	fmt.Printf("\n\033[90m[ID %d]\033[0m", res.CfgID)
	switch {
	case res.Info.Code >= 200 && res.Info.Code < 300:
		fmt.Printf("\n\033[32m[HTTP %d: %s]\033[0m",
			res.Info.Code, res.Info.Message)
	case res.Info.Code >= 300 && res.Info.Code < 400:
		fmt.Printf("\n\033[33m[HTTP %d: %s]\033[0m",
			res.Info.Code, res.Info.Message)
	case res.Info.Code >= 400 && res.Info.Code < 600:
		fmt.Printf("\n\033[31m[HTTP %d: %s]\033[0m",
			res.Info.Code, res.Info.Message)
	case res.Info.Code == 0 && res.Info.ConfigType == "grpc":
		fmt.Printf("\n\033[32m[GRPC %d: %s]\033[0m",
			res.Info.Code, res.Info.Message)
	case res.Info.Code != 0 && res.Info.ConfigType == "grpc":
		fmt.Printf("\n\033[31m[GRPC %d: %s]\033[0m",
			res.Info.Code, res.Info.Message)
	default:
		fmt.Printf("\n\033[31m[NOP %d: %s]\033[0m",
			res.Info.Code, res.Info.Message)
	}

	if len(res.Raw) == 0 {
		fmt.Printf("\n\033[90m[Empty body]\033[0m")
		return nil
	}

	if res.IsJSON {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, res.Raw, "", "  "); err != nil {
			return fmt.Errorf("%s: indent: %w", op, err)
		}
		fmt.Printf("\n[JSON Response]\n%s\n", prettyJSON.String())
	} else {
		if len(res.Raw) > 1024 {
			res.Raw = res.Raw[:1024]
			res.Raw = append(res.Raw, []byte("(truncated)")...)
		}
		fmt.Printf("\n[Raw Response]\n%s\n", res.Raw)
	}

	return nil
}
