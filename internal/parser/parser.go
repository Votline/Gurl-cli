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

type instruction struct {
	tID   int
	start int
	end   int
	key   string
}

func ParseStream(sData *[]gscan.Data, yield func(config.Config)) error {
	const op = "parser.parseStream"
	n := len(*sData)
	insts := [][]byte{[]byte("RESPONSE id=")}
	instsPos := make([]instruction, 0, 6)

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

	absEnd := 0
	cache := make([]config.Config, n)
	for i, d := range *sData {
		var cfg config.Config
		var execCfg config.Config
		instsPos = instsPos[:0]

		tID := targets[i]
		if tID != -1 {
			orig := cache[tID]
			if orig == nil {
				return fmt.Errorf("%s: cfg №[%d] target id not found", op, i)
			}
			execCfg = orig.Clone()

			var rep config.Config
			tp := "repeat"
			handleType(&rep, &tp, &d)
			rep.SetID(tID)
			rep.SetOrig(execCfg)

			cfg = rep
		} else {
			tp := fastExtract(d.RawData, &d.Entries, []byte("Type"))
			if tp == "" {
				return fmt.Errorf("%s: no config type", op)
			} else {
				if err := handleType(&cfg, &tp, &d); err != nil {
					return fmt.Errorf("%s: cfg №[%d] failed: %w", op, i, err)
				}
			}
			execCfg = cfg
		}

		tagOverhead := (len(d.Name)+2) + (len(d.Name)+3)+2
		localEnd := len(d.RawData)+tagOverhead
		absEnd += localEnd

		execCfg.SetID(i)
		cfg.SetEnd(absEnd)

		tID, err := handleInstructions(&d, &insts, func(inst instruction) {
			instsPos = append(instsPos, inst)
		})
		if err != nil {
			return fmt.Errorf("%s: check instr. execCfg's №[%d]: %w", op, i, err)
		}
		if tID != -1 && tID < n {
			for _, inst := range instsPos {
				if inst.tID < n {
					execCfg.SetDependency(config.Dependency{
						TargetID: inst.tID, Key: inst.key, Start: inst.start, End: inst.end})
				}
			}
		}

		if (needed[i/64] & (1 << (i % 64))) != 0 {
			cache[i] = execCfg.Clone()
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
		tID := fastExtract(d.RawData, &d.Entries, []byte("Target_ID"))
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

func handleInstructions(d *gscan.Data, insts *[][]byte, yield func(inst instruction)) (int, error) {
	const op = "parser.handleInstructions"

	start := bytes.IndexByte(d.RawData, '{')
	if start == -1 {
		return -1, nil
	}
	end := bytes.IndexByte(d.RawData, '}')
	if end == -1 {
		return -1, nil
	}

	for _, inst := range *insts {
		pIdx := bytes.Index(d.RawData[start+1:], inst)
		if pIdx == -1 {
			return -1, nil
		}
		pIdx += start

		idOffset := bytes.Index(d.RawData[pIdx:], []byte("id="))
		if idOffset == -1 {
			return -1, fmt.Errorf("%s: instruction %q: no id",
				op, string(inst))
		}
		idOffset += pIdx

		valStart := idOffset + 3
		for valStart < len(d.RawData) && isSpace(d.RawData[valStart]) {
			valStart++
		}

		valEnd := valStart
		for valEnd < len(d.RawData) && !isSpace(d.RawData[valEnd]) && d.RawData[valEnd] != '}' {
			valEnd++
		}

		if valStart == valEnd {
			return -1, fmt.Errorf("%s: empty id value", op)
		}

		end = bytes.IndexByte(d.RawData[valEnd:], '}') + valEnd + 1

		tID := atoi(d.RawData[valStart:valEnd])
		if tID == -1 {
			return -1, fmt.Errorf("%s: invalid id %q", op, d.RawData[valStart:valEnd])
		}

		instKey := ""
		for _, ent := range d.Entries {
			if pIdx >= ent.ValStart && pIdx <= ent.ValEnd {
				start = pIdx - ent.ValStart
				end = end - ent.ValStart
				instKey = unsafe.String(unsafe.SliceData(d.RawData[ent.KeyStart:ent.KeyEnd]), ent.KeyEnd-ent.KeyStart)
			}
		}

		yield(instruction{
			tID:   tID,
			start: start,
			end:   end,
			key:   instKey,
		})
	}

	return 0, nil
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
