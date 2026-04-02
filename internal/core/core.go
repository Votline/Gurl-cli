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

	"github.com/Votline/Gurl-cli/internal/buffer"
	"github.com/Votline/Gurl-cli/internal/config"
	"github.com/Votline/Gurl-cli/internal/parser"
	"github.com/Votline/Gurl-cli/internal/transport"

	"github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
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

func Start(cType, cPath string, cCreate, disablePrint bool, log *zap.Logger) error {
	if cCreate {
		return config.Create(cType, cPath)
	}
	config.Init()
	vars := make(map[string][]byte)
	return handleConfig(cPath, disablePrint, vars, log)
}

func handleConfig(cPath string, disablePrint bool, vars map[string][]byte, log *zap.Logger) error {
	const op = "core.handleConfig"

	var sData []gscan.Data
	soloCfg := false

	if _, err := os.Stat(cPath); err == nil {
		sData, err = gurlf.ScanFile(cPath)
		if err != nil {
			return fmt.Errorf("%s: scan file %q: %w", op, cPath, err)
		}
	} else {
		oneCfg := unsafe.Slice(unsafe.StringData(cPath), len(cPath))
		sData, err = gurlf.Scan(oneCfg)
		if err != nil {
			return fmt.Errorf("%s: scan %q: %w", op, cPath, err)
		}
		soloCfg = true
	}

	resHub := make([]*transport.Result, 0, len(sData))
	cfgFileRBuf := buffer.NewRb[config.Config]()
	parserRBuf := buffer.NewRb[config.Config]()
	transportRBuf := buffer.NewRb[*transport.Result]()
	resPrintBuf := buffer.NewRb[*transport.Result]()
	trnsp := transport.NewTransport(transportRBuf.Write, log)

	if soloCfg {
		cfgFileRBuf = buffer.NewNop[config.Config]()
	}

	if disablePrint {
		resPrintBuf = buffer.NewNop[*transport.Result]()
	}

	isCrashed := false
	var globalErr error
	var wg sync.WaitGroup
	wg.Go(func() {
		defer cfgFileRBuf.Close()
		defer resPrintBuf.Close()

		for {
			cfg := parserRBuf.Read()
			if cfg == nil {
				break
			}

			for {
				cfgToFile := cfg.Clone()
				log.Debug("processing config",
					zap.String("op", op),
					zap.String("name", cfg.GetName()),
					zap.Int("id", cfg.GetID()))

				applyDeps(cfg, &resHub, vars, log)

				res := transportRBuf.Read()
				execCfg := cfg.UnwrapExec()

				if ok := applyVars(cfg, vars, log); !ok {
					break
				}

				if ok := applyEnvs(cfg, log); !ok {
					break
				}

				applyIgnrCrt(cfg, execCfg, log)

				applyWait(cfg, execCfg, log)

				if impCfg, ok := cfg.(*config.ImportConfig); ok {
					log.Debug("import config",
						zap.String("op", op),
						zap.String("name", cfg.GetName()),
						zap.Int("id", cfg.GetID()))

					if err := handleConfig(impCfg.TargetPath, disablePrint, vars, log); err != nil {
						log.Error("Failed to handle config",
							zap.String("op", op),
							zap.String("name", cfg.GetName()),
							zap.Int("id", cfg.GetID()),
							zap.Error(err))
						globalErr = err
						isCrashed = true // for 'copyTail'
						return
					}
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
					cfgFileRBuf.Write(cfgToFile)
				}

				if id == parser.ExpectDone && !isCrashed {
					break
				} else if id == parser.ExpectDone && isCrashed {
					cfg.Release()
					transportRBuf.Write(res)
					return
				}

				if id == parser.ExpectCrash {
					cfg.Release()
					transportRBuf.Write(res)
					isCrashed = true
					return
				}

				if isCrashed {
					cfg.Release()
					transportRBuf.Write(res)
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
					transportRBuf.Write(res)
					break
				}

				cfg.Release()
				transportRBuf.Write(res)
				cfg = nextCfg
				cfg.SetEnd(origEnd)
			}
		}
	})

	if !soloCfg {
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
				cfg = cfgFileRBuf.Read()
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
	}

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

	if err := parser.ParseStream(&sData, parserRBuf.Write, log); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	parserRBuf.Close()

	wg.Wait()

	if globalErr != nil {
		return fmt.Errorf("%s: %w", op, globalErr)
	}
	return nil
}

func applyDeps(cfg config.Config, resHub *[]*transport.Result, vars map[string][]byte, log *zap.Logger) {
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

		switch d.TargetID {
		case config.DataFromFile:
			cfg.SetFlag(config.FlagUseFileCookies)
			cfg.Apply(d.Start, d.End, d.Key, nil)
			continue
		case config.RandomData:
			rawSnapshot := make([]byte, len(cfg.GetRaw(d.Key)))
			copy(rawSnapshot, cfg.GetRaw(d.Key))

			var instructionBytes []byte
			if !getInstructionBytes(cfg, d, &instructionBytes, log) {
				continue
			}

			val := make([]byte, 36)
			parser.ParseRandom(instructionBytes, &val)
			if val == nil {
				log.Error("Failed to parse random",
					zap.String("op", op),
					zap.String("key", d.Key),
					zap.String("inst", string(instructionBytes)))
				continue
			}

			log.Debug("apply random",
				zap.String("op", op),
				zap.String("name", cfg.GetName()),
				zap.String("key", d.Key),
				zap.String("val", unsafe.String(unsafe.SliceData(val), len(val))))

			cfg.Apply(d.Start, d.End, d.Key, val)

			continue
		case config.DataFromVariable:
			rawSnapshot := make([]byte, len(cfg.GetRaw(d.Key)))
			copy(rawSnapshot, cfg.GetRaw(d.Key))

			var instructionBytes []byte
			if !getInstructionBytes(cfg, d, &instructionBytes, log) {
				continue
			}

			var key []byte
			parser.GetVarKey(instructionBytes, &key)
			if key == nil {
				log.Error("Failed to get variable key",
					zap.String("op", op),
					zap.String("key", d.Key),
					zap.String("inst", string(instructionBytes)))
				continue
			}

			keyStr := unsafe.String(unsafe.SliceData(key), len(key))
			val := vars[keyStr]
			if val == nil {
				log.Error("Failed to get variable",
					zap.String("op", op),
					zap.String("key", keyStr))
				continue
			}

			cfg.Apply(d.Start, d.End, d.Key, val)

			log.Debug("apply variable",
				zap.String("op", op),
				zap.String("name", cfg.GetName()),
				zap.Int("id", cfg.GetID()),
				zap.String("key", d.Key),
				zap.String("val", unsafe.String(unsafe.SliceData(val), len(val))))

			continue
		case config.DataFromEnvironment:
			rawSnapshot := make([]byte, len(cfg.GetRaw(d.Key)))
			copy(rawSnapshot, cfg.GetRaw(d.Key))

			var from []byte
			if ok := getInstructionBytes(cfg, d, &from, log); !ok {
				continue
			}

			var key []byte
			parser.ParseEnv(&from, &key)

			if key == nil || from == nil {
				log.Error("Failed to get environment key",
					zap.String("op", op),
					zap.String("key", unsafe.String(unsafe.SliceData(key), len(key))),
					zap.String("from", unsafe.String(unsafe.SliceData(from), len(from))))
				continue
			}

			val := from
			keyStr := unsafe.String(unsafe.SliceData(key), len(key))
			fromStr := unsafe.String(unsafe.SliceData(from), len(from))
			if bytes.Equal(from, []byte("os")) {
				valStr := os.Getenv(keyStr)
				if valStr == "" {
					log.Error("Failed to get environment",
						zap.String("op", op),
						zap.String("key", keyStr),
						zap.String("from", unsafe.String(unsafe.SliceData(from), len(from))))
					continue
				}
				val = unsafe.Slice(unsafe.StringData(valStr), len(valStr))
			} else {
				path := fromStr
				file, err := os.Open(path)
				if err != nil {
					log.Error("Failed to open file",
						zap.String("op", op),
						zap.String("key", keyStr),
						zap.String("from", unsafe.String(unsafe.SliceData(from), len(from))),
						zap.Error(err))
					continue
				}
				defer file.Close()

				data, err := os.ReadFile(path)
				if err != nil {
					log.Error("Failed to read file",
						zap.String("op", op),
						zap.String("key", keyStr),
						zap.String("from", unsafe.String(unsafe.SliceData(from), len(from))),
						zap.Error(err))
					continue
				}

				parser.SearchKey(data, key, &val)

				if val == nil {
					log.Error("Failed to get environment",
						zap.String("op", op),
						zap.String("key", keyStr),
						zap.String("from", unsafe.String(unsafe.SliceData(from), len(from))))
					continue
				}
			}

			cfg.Apply(d.Start, d.End, d.Key, val)
			log.Debug("apply environment",
				zap.String("op", op),
				zap.String("name", cfg.GetName()),
				zap.Int("id", cfg.GetID()),
				zap.String("key", keyStr),
				zap.String("from", unsafe.String(unsafe.SliceData(from), len(from))),
				zap.String("val", unsafe.String(unsafe.SliceData(val), len(val))))

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

		var instructionBytes []byte
		if !getInstructionBytes(cfg, d, &instructionBytes, log) {
			continue
		}

		bind.To(cfg, d.Start, d.End, d.Key, val, instructionBytes)

		log.Debug("applied dependencies",
			zap.String("op", op),
			zap.String("name", cfg.GetName()),
			zap.String("key", d.Key),
			zap.String("inst", unsafe.String(unsafe.SliceData(instructionBytes), len(instructionBytes))),
			zap.String("val", unsafe.String(unsafe.SliceData(val), len(val))))
	}
}

func applyVars(cfg config.Config, vars map[string][]byte, log *zap.Logger) bool {
	const op = "core.applyVars"

	gscanVars, err := gurlf.Scan(cfg.GetVars())
	if err != nil {
		log.Error("Failed to scan vars",
			zap.String("op", op),
			zap.String("name", cfg.GetName()),
			zap.Int("id", cfg.GetID()),
			zap.Error(err))
		return false
	}
	parser.ApplyVars(gscanVars, vars)

	log.Debug("set variables",
		zap.String("op", op),
		zap.String("name", cfg.GetName()),
		zap.Int("id", cfg.GetID()),
		zap.String("vars", unsafe.String(unsafe.SliceData(cfg.GetVars()), len(cfg.GetVars()))))

	return true
}

func applyEnvs(cfg config.Config, log *zap.Logger) bool {
	const op = "core.applyEnvs"

	gscanEnvs, err := gurlf.Scan(cfg.GetEnvs())
	if err != nil {
		log.Error("Failed to scan envs",
			zap.String("op", op),
			zap.String("name", cfg.GetName()),
			zap.Int("id", cfg.GetID()),
			zap.Error(err))
		return false
	}
	parser.ApplyEnvs(gscanEnvs)

	log.Debug("set environments",
		zap.String("op", op),
		zap.String("name", cfg.GetName()),
		zap.Int("id", cfg.GetID()),
		zap.String("envs", unsafe.String(unsafe.SliceData(cfg.GetEnvs()), len(cfg.GetEnvs()))))

	return true
}

func applyIgnrCrt(cfg, execCfg config.Config, log *zap.Logger) {
	const op = "core.applyIgnrCrt"

	orig := cfg.GetIgnrCrt()

	if orig != nil {
		execCfg.SetIgnrCrt(orig)
	}

	log.Debug("IgnoreCert",
		zap.String("op", op),
		zap.String("name", execCfg.GetName()),
		zap.String("ignrCrt", unsafe.String(unsafe.SliceData(orig), len(orig))))
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
			zap.String("url", unsafe.String(unsafe.SliceData(v.URL), len(v.URL))),
			zap.String("method", unsafe.String(unsafe.SliceData(v.Method), len(v.Method))),
			zap.String("body", unsafe.String(unsafe.SliceData(v.Body), len(v.Body))),
			zap.String("headers", unsafe.String(unsafe.SliceData(v.Headers), len(v.Headers))),
			zap.String("cookie", unsafe.String(unsafe.SliceData(v.CookieIn), len(v.CookieIn))))

	case *config.GRPCConfig:
		log.Debug("send grpc",
			zap.String("op", op),
			zap.String("target", unsafe.String(unsafe.SliceData(v.Target), len(v.Target))),
			zap.String("endpoint", unsafe.String(unsafe.SliceData(v.Endpoint), len(v.Endpoint))),
			zap.String("body", unsafe.String(unsafe.SliceData(v.Data), len(v.Data))),
			zap.String("protoPath", unsafe.String(unsafe.SliceData(v.ProtoPath), len(v.ProtoPath))),
			zap.String("importPaths", unsafe.String(unsafe.SliceData(v.ImportPaths), len(v.ImportPaths))),
			zap.String("dialOpts", unsafe.String(unsafe.SliceData(v.DialOpts), len(v.DialOpts))))

		err = trnsp.DoGRPC(v, res)
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

func getInstructionBytes(cfg config.Config, d config.Dependency, buf *[]byte, log *zap.Logger) bool {
	const op = "core.getInstructionBytes"

	rawSnapshot := make([]byte, len(cfg.GetRaw(d.Key)))
	copy(rawSnapshot, cfg.GetRaw(d.Key))

	if d.Start > len(rawSnapshot) || d.End > len(rawSnapshot) {
		log.Error("Invalid instruction",
			zap.String("op", op),
			zap.Int("id", cfg.GetID()),
			zap.String("name", cfg.GetName()),
			zap.String("key", d.Key),
			zap.Int("start", d.Start),
			zap.Int("end", d.End),
			zap.Int("len", len(rawSnapshot)))
		return false
	}

	*buf = rawSnapshot[d.Start:d.End]

	return true
}
