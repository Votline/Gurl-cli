package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"unsafe"

	"gcli/internal/config"

	"github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
	"go.uber.org/zap"
)

type instruction struct {
	tID   int
	start int
	end   int
	key   string
	insTp string
}

func ParseStream(sData *[]gscan.Data, yield func(config.Config), log *zap.Logger) error {
	const op = "parser.parseStream"
	n := len(*sData)
	insts := [][]byte{[]byte("RESPONSE id="), []byte("COOKIES id=")}
	instsPos := make([]instruction, 0, 6)

	log.Debug("preparing configs",
		zap.String("op", op),
		zap.Int("count", n))

	targets := make([]int, n)
	needed := make([]uint64, (n/64)+1)
	for i, d := range *sData {
		tID, err := handleRepeat(&d)
		if err != nil {
			log.Debug("check cfg failed",
				zap.String("op", op),
				zap.String("name", string(d.Name)),
				zap.String("raw", string(d.RawData)))

			return fmt.Errorf("%s: check cfg №[%d] failed: %w", op, i, err)
		}
		targets[i] = tID
		if tID != -1 && tID < n {
			needed[tID/64] |= (1 << (tID % 64))
		}
	}

	log.Debug("processing configs",
		zap.String("op", op),
		zap.Int("count", n))

	absEnd := 0
	cache := make([]config.Config, n)
	for i, d := range *sData {
		var cfg config.Config
		var execCfg config.Config
		instsPos = instsPos[:0]

		log.Debug("processing config",
			zap.String("op", op),
			zap.String("name", string(d.Name)),
			zap.Int("id", i))

		tID := targets[i]
		if tID != config.NoRepeatConfig {

			if tID > n || tID > len(cache) {
				log.Error("invalid repeat target id",
					zap.String("op", op),
					zap.String("name", string(d.Name)),
					zap.Int("id", tID))
				return fmt.Errorf("%s: invalid repeat target id", op)
			}

			log.Debug("alloc repeat config",
				zap.String("op", op),
				zap.String("name", string(d.Name)),
				zap.Int("id", tID))

			execCfg = config.Alloc(cache[tID])
			if execCfg == nil {
				log.Debug("repeat config not found",
					zap.String("op", op),
					zap.String("name", string(d.Name)),
					zap.String("raw", string(d.RawData)))

				return fmt.Errorf("%s: cfg №[%d] target id not found", op, i)
			}

			var rep config.Config
			tp := "repeat"
			handleType(&rep, &tp, &d)
			rep.(*config.RepeatConfig).SetTargetID(tID)
			rep.(*config.RepeatConfig).Orig = execCfg

			cfg = rep

			nD, err := getReplaceData(rep.(*config.RepeatConfig))
			if err != nil {
				return fmt.Errorf("%s: get replace data: %w", op, err)
			}
			d = nD
		} else {
			tp := fastExtract(d.RawData, &d.Entries, []byte("Type"))

			if tp == "" {
				log.Debug("no config type",
					zap.String("op", op),
					zap.String("name", string(d.Name)),
					zap.String("raw", string(d.RawData)))
				return fmt.Errorf("%s: no config type", op)
			} else {
				if err := handleType(&cfg, &tp, &d); err != nil {
					log.Debug("check cfg failed",
						zap.String("op", op),
						zap.String("name", string(d.Name)),
						zap.String("raw", string(d.RawData)))
					return fmt.Errorf("%s: cfg №[%d] failed: %w", op, i, err)
				}
			}
			execCfg = cfg
		}

		if err := handleInstructions(&d, &insts, func(inst instruction) {
			instsPos = append(instsPos, inst)
		}); err != nil {
			log.Debug("check instr. execCfg failed",
				zap.String("op", op),
				zap.String("name", string(d.Name)),
				zap.String("raw", string(d.RawData)))
			return fmt.Errorf("%s: check instruction execCfg №[%d]: %w",
				op, i, err)
		}
		for _, inst := range instsPos {
			if inst.tID > n {
				log.Error("invalid instruction target id",
					zap.String("op", op),
					zap.String("name", string(d.Name)),
					zap.Int("id", inst.tID))
				return fmt.Errorf("%s: invalid instruction target id", op)
			}

			execCfg.SetDependency(config.Dependency{
				TargetID: inst.tID, Key: inst.key, Start: inst.start, End: inst.end, InsTp: inst.insTp,
			})
			log.Debug("set dependency",
				zap.String("op", op),
				zap.String("name", string(d.Name)),
				zap.Int("id", i),
				zap.Int("target", inst.tID),
				zap.String("key", inst.key),
				zap.Int("start", inst.start),
				zap.Int("end", inst.end),
				zap.String("type", inst.insTp))
		}

		tagOverhead := (len(d.Name) + 2) + (len(d.Name) + 3) + 2
		localEnd := len(d.RawData) + tagOverhead
		absEnd += localEnd

		cfg.SetID(i)
		cfg.SetEnd(absEnd)

		log.Debug("set config end",
			zap.String("op", op),
			zap.String("name", string(d.Name)),
			zap.Int("id", i),
			zap.Int("end", absEnd))

		if r, ok := cfg.(*config.RepeatConfig); ok {
			if err := applyReplace(r); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}

		if (needed[i/64] & (1 << (i % 64))) != 0 {
			nw := config.Alloc(execCfg)
			cache[i] = nw
		}

		yield(cfg)
	}

	return nil
}

func handleRepeat(d *gscan.Data) (int, error) {
	const op = "parser.handleRepeat"

	tp := fastExtract(d.RawData, &d.Entries, []byte("Type"))
	if tp == "" {
		return -1, fmt.Errorf("%s: no config type", op)
	}
	if tp == "repeat" {
		tID := fastExtract(d.RawData, &d.Entries, []byte("TargetID"))
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

func handleInstructions(d *gscan.Data, insts *[][]byte, yield func(inst instruction)) error {
	const op = "parser.handleInstructions"

	start := bytes.IndexByte(d.RawData, '{')
	if start == -1 {
		return nil
	}
	end := bytes.IndexByte(d.RawData, '}')
	if end == -1 {
		return nil
	}

	for _, inst := range *insts {
		curOffset := start + 1
		for {
			pIdx := bytes.Index(d.RawData[curOffset:], inst)
			if pIdx == -1 {
				break
			}
			pIdx += curOffset

			idOffset := bytes.Index(d.RawData[pIdx:], []byte("id="))
			if idOffset == -1 {
				return fmt.Errorf("%s: instruction %q: no id",
					op, string(inst))
			}
			idOffset += pIdx

			instTp := unsafe.String(unsafe.SliceData(d.RawData[pIdx:idOffset-1]), len(d.RawData[pIdx:idOffset-1]))

			valStart := idOffset + 3
			for valStart < len(d.RawData) && isSpace(d.RawData[valStart]) {
				valStart++
			}

			valEnd := valStart
			for valEnd < len(d.RawData) && !isSpace(d.RawData[valEnd]) && d.RawData[valEnd] != '}' {
				valEnd++
			}

			if valStart == valEnd {
				return fmt.Errorf("%s: empty id value", op)
			}

			end = bytes.IndexByte(d.RawData[pIdx:], '}')
			if end == -1 {
				return fmt.Errorf("%s: instruction %q: no end", op, string(inst))
			}
			absEnd := pIdx + end + 1 // catch '}'

			absStart := pIdx
			for absStart > 0 && d.RawData[absStart] != '{' {
				absStart--
			}

			tID := -1
			args := d.RawData[valStart:valEnd]
			if bytes.Equal(args, []byte("file")) {
				tID = config.DataFromFile
			} else {
				tID = atoi(d.RawData[valStart:valEnd])
				if tID == -1 {
					return fmt.Errorf("%s: invalid id %q", op, d.RawData[valStart:valEnd])
				}
			}

			localStart := start
			localEnd := absEnd
			instKey := ""
			for _, ent := range d.Entries {
				if pIdx >= ent.ValStart && pIdx <= ent.ValEnd {
					localStart = absStart - ent.ValStart
					localEnd = absEnd - ent.ValStart
					instKey = unsafe.String(unsafe.SliceData(d.RawData[ent.KeyStart:ent.KeyEnd]), ent.KeyEnd-ent.KeyStart)
				}
			}

			yield(instruction{
				tID:   tID,
				start: localStart,
				end:   localEnd,
				key:   instKey,
				insTp: instTp,
			})
			curOffset = absEnd
		}
	}

	return nil
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
	case "repeat":
		obj, itab := config.GetRepeat()
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

func fastExtract(data []byte, ents *[]gscan.Entry, need []byte) string {
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

func atoi(data []byte) int {
	res := 0
	foundDigit := false
	cur := 0
	for cur < len(data) && data[cur] >= '0' && data[cur] <= '9' {
		res = res*10 + int(data[cur]-'0')
		foundDigit = true
		cur++
	}

	if !foundDigit {
		return -1
	}

	return res
}

func applyReplace(r *config.RepeatConfig) error {
	const op = "parser.applyReplace"

	if len(r.Replace) == 0 {
		return nil
	}

	sData, err := gurlf.Scan(r.Replace)
	if err != nil {
		return fmt.Errorf("%s: scan replace: %w", op, err)
	}

	for _, d := range sData {
		if len(d.RawData) == 0 {
			continue
		}

		for _, ent := range d.Entries {
			if ent.ValEnd == 0 {
				continue
			}

			key := unsafe.String(unsafe.SliceData(d.RawData[ent.KeyStart:ent.KeyEnd]), ent.KeyEnd-ent.KeyStart)
			val := d.RawData[ent.ValStart:ent.ValEnd]

			r.Orig.Apply(0, config.MaxLen, key, val)
		}

	}
	return nil
}

func getReplaceData(r *config.RepeatConfig) (gscan.Data, error) {
	const op = "parser.getReplaceData"
	var zero gscan.Data

	sData, err := gurlf.Scan(r.Replace)
	if err != nil {
		return zero, fmt.Errorf("%s: scan replace: %w", op, err)
	}

	if len(sData) == 0 {
		return zero, nil
	}

	return sData[0], nil
}
