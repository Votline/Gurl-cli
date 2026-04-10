// Package parser parser.go turns data from scanner to config objects.
// It also set config dependencies.
package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/Votline/Gurl-cli/internal/config"

	"github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
	"go.uber.org/zap"
)

// insts is a list of instruction names.
// Its like a macroses in C.
var insts = [][]byte{
	[]byte("RESPONSE"),
	[]byte("COOKIES"),
	[]byte("RANDOM"),
	[]byte("VARIABLE"),
	[]byte("ENVIRONMENT"),
}

// markers is a list of markers for config data.
// Its used for instructions.
var markers = [][]byte{
	[]byte("id="),
	[]byte("oneof="),
	[]byte("key="),
	[]byte("from="),
}

// ParseStream accepts result of scanner and call yield for each config.
// It also set config dependencies
// And replces 'replace' config type with target config
// And replaces config data with data from replace field.
func ParseStream(sData *[]gscan.Data, yield func(config.Config), log *zap.Logger) error {
	const op = "parser.parseStream"
	n := len(*sData)
	instsPos := make([]config.Dependency, 0, 6)

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

		if err := handleInstructions(&d, insts, func(inst config.Dependency) {
			instsPos = append(instsPos, inst)
		}); err != nil {
			log.Error("check instruction execCfg failed",
				zap.String("op", op),
				zap.String("name", string(d.Name)),
				zap.Int("id", i))
			return fmt.Errorf("%s: check instruction execCfg №[%d]: %w",
				op, i, err)
		}
		for _, inst := range instsPos {
			if inst.TargetID > n {
				log.Error("invalid instruction target id",
					zap.String("op", op),
					zap.String("name", string(d.Name)),
					zap.Int("id", inst.TargetID))
				return fmt.Errorf("%s: invalid instruction target id", op)
			}

			execCfg.SetDependency(config.Dependency{
				TargetID: inst.TargetID, Key: inst.Key, Start: inst.Start, End: inst.End, InsTp: inst.InsTp,
			})
			log.Debug("set dependency",
				zap.String("op", op),
				zap.String("name", string(d.Name)),
				zap.Int("id", i),
				zap.Int("target", inst.TargetID),
				zap.String("key", inst.Key),
				zap.Int("start", inst.Start),
				zap.Int("end", inst.End),
				zap.String("type", inst.InsTp))
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

// handleRepeat extracts target id from 'repeat' config.
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

// handleInstructions extracts dependencies from config.
// Accepts config and list of instruction names.
// Calls yield for each dependency.
func handleInstructions(d *gscan.Data, insts [][]byte, yield func(inst config.Dependency)) error {
	const op = "parser.handleInstructions"

	start := bytes.IndexByte(d.RawData, '{')
	if start == -1 {
		return nil
	}
	end := bytes.IndexByte(d.RawData, '}')
	if end == -1 {
		return nil
	}

	for _, inst := range insts {
		curOffset := start + 1

		for {
			pIdx := bytes.Index(d.RawData[curOffset:], inst)
			if pIdx == -1 {
				break
			}
			pIdx += curOffset

			depType := 0
			minIdx := -1
			markerLen := 0
			for _, m := range markers {
				idx := bytes.Index(d.RawData[pIdx:], m)
				if idx != -1 {
					if minIdx == -1 || idx < minIdx {
						minIdx = idx
						markerLen = len(m)
					}
				}
			}
			if minIdx == -1 {
				return fmt.Errorf("%s: instruction %q: no id",
					op, string(inst))
			}
			depType += pIdx + minIdx

			typeEnd := depType
			for typeEnd > pIdx && isSpace(d.RawData[typeEnd-1]) {
				typeEnd--
			}
			instTp := unsafe.String(unsafe.SliceData(d.RawData[pIdx:typeEnd]), typeEnd-pIdx)

			valStart := depType + markerLen
			for valStart < len(d.RawData) && isSpace(d.RawData[valStart]) {
				valStart++
			}

			valEnd := valStart
			for valEnd < len(d.RawData) && !isSpace(d.RawData[valEnd]) && d.RawData[valEnd] != '}' {
				valEnd++
			}
			// valEnd++ // catch '}'

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

			switch {
			case instTp == "RANDOM":
				tID = config.RandomData
			case bytes.Equal(args, []byte("file")):
				tID = config.DataFromFile
			case instTp == "VARIABLE":
				tID = config.DataFromVariable
			case instTp == "ENVIRONMENT":
				tID = config.DataFromEnvironment
			default:
				tID = atoi(d.RawData[valStart:valEnd])
				if tID == -1 {
					return fmt.Errorf("%s: invalid id %q in instruction type %q", op, d.RawData[valStart:valEnd], instTp)
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

			yield(config.Dependency{
				TargetID: tID,
				Start:    localStart,
				End:      localEnd,
				Key:      instKey,
				InsTp:    instTp,
			})
			curOffset = absEnd
		}
	}

	return nil
}

// handleType creates config object and unmrashal it.
// Used pre-allocated config buffers to zero allocations.
// Accepts config objet, config type and config data.
// Into config object unmarshals config data.
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
	case "import":
		obj, itab := config.GetImport()
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

// ParseFindConfig accepts result of scanner and config object.
// Its like a ParseStream but for one config.
// It also set config dependencies
// And replces 'repeat' config type with target config
// And replaces config data with data from replace field.
func ParseFindConfig(sData *[]gscan.Data, cfg *config.Config, tID int) error {
	const op = "parser.ParseFindConfig"

	if tID < 0 || tID >= len(*sData) {
		return fmt.Errorf("%s: config №[%d] not found", op, tID)
	}

	d := (*sData)[tID]

	tp := fastExtract(d.RawData, &d.Entries, []byte("Type"))
	if tp == "" {
		return fmt.Errorf("%s: no config type", op)
	}

	if tp == "repeat" {
		targetIDStr := fastExtract(d.RawData, &d.Entries, []byte("TargetID"))
		if targetIDStr == "" {
			return fmt.Errorf("%s: no target id", op)
		}

		targetIDBytes := unsafe.Slice(unsafe.StringData(targetIDStr), len(targetIDStr))
		parentID := atoi(targetIDBytes)

		var parentCfg config.Config
		if err := ParseFindConfig(sData, cfg, parentID); err != nil {
			return fmt.Errorf("%s: failed to find parent config: %w", op, err)
		}

		*cfg = config.Alloc(parentCfg)

		if err := handleType(cfg, &tp, &d); err != nil {
			return fmt.Errorf("%s: failed to handle type: %w", op, err)
		}
	} else {
		if err := handleType(cfg, &tp, &d); err != nil {
			return fmt.Errorf("%s: failed to handle type: %w", op, err)
		}
	}

	if err := handleInstructions(&d, insts, func(inst config.Dependency) {
		(*cfg).SetDependency(config.Dependency{
			TargetID: inst.TargetID, Key: inst.Key, Start: inst.Start, End: inst.End, InsTp: inst.InsTp,
		})
	}); err != nil {
		return fmt.Errorf("%s: failed to handle instructions: %w", op, err)
	}

	(*cfg).SetID(tID)

	return nil
}

// applyReplace replaces config data with data from replace field.
// Changes will occur in original config object.
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

// getReplaceData extracts data from replace field.
// Accepts repeat config object, return scanner data and error.
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
