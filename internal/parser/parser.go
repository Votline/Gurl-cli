package parser

import (
	"bytes"
	"fmt"
	"gcli/internal/config"
	"strconv"
	"unsafe"

	"github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
)

func Parse(cPath string, yield func(config.Config) error) error {
	const op = "parser.Parse"

	config.Init()

	sData, err := gurlf.ScanFile(cPath)
	if err != nil {
		return fmt.Errorf("%s: scan file %q: %w", op, cPath, err)
	}

	if err := parseStream(&sData, yield); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func parseStream(sData *[]gscan.Data, yield func(config.Config) error) error {
	const op = "parser.parseStream"
	n := len(*sData)

	targets := make([]int, n)
	needed := make([]uint64, (n/64)+1)
	for i, d := range *sData {
		tID, err := handleRepeat(&d)
		if err != nil {
			return fmt.Errorf("%s: check cfg №[%d] failed: %w", op, i, err)
		}
		targets[i] = tID
		if tID != -1 && tID < n {
			needed[tID/64] |= (1 << (tID % 64))
		}
	}

	cache := make(map[int]config.Config)
	for i, d := range *sData {
		var cfg config.Config

		tID := targets[i]
		if tID != -1 {
			orig, ok := cache[tID]
			if !ok {
				return fmt.Errorf("%s: cfg №[%d] target id not found", op, i)
			}
			cfg = orig.Clone()
		} else {
			tp := fastExtract(&d.RawData, &d.Entries, []byte("type"))
			if tp == "" {
				return fmt.Errorf("%s: no config type", op)
			} else {
				if err := handleType(&cfg, &tp, &d); err != nil {
					return fmt.Errorf("%s: cfg №[%d] failed: %w", op, i, err)
				}
			}
		}
		cfg.SetID(i)

		if (needed[i/64] & (1 << (i % 64))) != 0 {
			cache[i] = cfg.Clone()
		}

		if err := yield(cfg); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func handleRepeat(d *gscan.Data) (int, error) {
	const op = "parser.handleRepeat"

	tp := fastExtract(&d.RawData, &d.Entries, []byte("type"))
	if tp == "" {
		return -1, fmt.Errorf("%s: no config type", op)
	}
	if tp == "repeat" {
		tID := fastExtract(&d.RawData, &d.Entries, []byte("target_id"))
		if tID == "" {
			return -1, fmt.Errorf("%s: no target id", op)
		}

		id, err := strconv.Atoi(tID)
		if err != nil {
			return -1, fmt.Errorf("%s: invalid target id: %w", op, err)
		}

		return id, nil
	}

	return -1, nil
}

func handleType(c *config.Config, tp *string, d *gscan.Data) error {
	const op = "parser.handleType"

	switch *tp {
	case "http":
		obj, itab := config.GetHTTP()
		*(*uintptr)(unsafe.Pointer(c)) = itab
		*(*uintptr)(unsafe.Add(unsafe.Pointer(c), uintptr(8))) = uintptr(unsafe.Pointer(obj))
	case "grpc":
		obj, itab := config.GetGRPC()
		*(*uintptr)(unsafe.Pointer(c)) = itab
		*(*uintptr)(unsafe.Add(unsafe.Pointer(c), uintptr(8))) = uintptr(unsafe.Pointer(obj))
	default:
		return fmt.Errorf("%s: undefined cfg type: %q", op, *tp)
	}

	if err := gurlf.Unmarshal(*d, *c); err != nil {
		return fmt.Errorf("%s: type %q: %w", op, *tp, err)
	}

	return nil
}

func fastExtract(raw *[]byte, ents *[]gscan.Entry, need []byte) string {
	data := *raw
	entries := *ents
	for _, ent := range entries {
		if bytes.Equal(data[ent.KeyStart:ent.KeyEnd], need) {
			vS, vE := ent.ValStart, ent.ValEnd
			tmp := data[vS:vE]
			tp := unsafe.String(unsafe.SliceData(tmp), len(tmp))
			return tp
		}
	}

	return ""
}
