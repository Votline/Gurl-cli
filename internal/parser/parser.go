package parser

import (
	"bytes"
	"fmt"
	"gcli/internal/config"
	"unsafe"

	"github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
)

func Parse(cPath string) ([]config.Config, error) {
	const op = "parser.Parse"

	sData, err := gurlf.ScanFile(cPath)
	if err != nil {
		return nil, fmt.Errorf("%s: scan file %q: %w", op, cPath, err)
	}

	cfgs, err := parseData(&sData)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return cfgs, nil
}

func parseData(sData *[]gscan.Data) ([]config.Config, error) {
	const op = "parser.parseData"

	cfgs := make([]config.Config, len(*sData))
	for i, d := range *sData {
		tp := fastExtractType(&d.RawData, &d.Entries)
		if tp == "" {
			return cfgs, fmt.Errorf("%s: no config type", op)
		}

		var cfg config.Config
		if err := handleType(&cfg, &tp, &d); err != nil {
			return cfgs, fmt.Errorf("%s: cfg №[%d] failed: %w",
				op, i, err)
		}

		cfg.SetID(i)
		if err := resolveRepeat(i, &cfg, &cfgs); err != nil {
			return cfgs, fmt.Errorf("%s: cfg №[%d] repeat failed: %w",
				op, i, err)
		}

		cfgs[i] = cfg
	}

	return cfgs, nil
}

func handleType(c *config.Config, tp *string, d *gscan.Data) error {
	const op = "parser.handleType"

	switch *tp {
	case "http": *c = &config.HTTPConfig{}
	case "grpc": *c = &config.GRPCConfig{}
	case "repeat": *c = &config.RepeatConfig{}
	default:
		return fmt.Errorf("%s: undefined cfg type: %q", op, *tp)
	}

	if err := gurlf.Unmarshal(*d, *c); err != nil {
		return fmt.Errorf("%s: type %q: %w", op, *tp, err)
	}

	return nil
}

func resolveRepeat(i int, cfg *config.Config, cfgs *[]config.Config) error {
	const op = "parser.resolveRepeat"

	r, ok := (*cfg).(*config.RepeatConfig)
	if !ok {
		return nil
	}

	if r.TargetID >= i || r.TargetID < 0 {
		return fmt.Errorf("%s: idx: invalid repeat idx: %d", op, r.TargetID)
	}

	(*cfg) = (*cfgs)[r.TargetID].Clone()
	(*cfg).SetID(i)

	return nil
}

func fastExtractType(raw *[]byte, ents *[]gscan.Entry) (string) {
	for _, ent := range *ents {
		if bytes.Equal( (*raw)[ent.KeyStart : ent.KeyEnd], []byte("type")) {
			vS, vE := ent.ValStart, ent.ValEnd
			tmp := (*raw)[vS:vE]
			tp := unsafe.String(unsafe.SliceData(tmp), len(tmp))
			return tp
		}
	}

	return ""
}
