package parser

import (
	"fmt"
	"gcli/internal/config"

	"github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
)

func Parse(cPath string) ([]config.Config, error) {
	const op = "parser.Parse"

	sData, err := gurlf.ScanFile(cPath)
	if err != nil {
		return nil, fmt.Errorf("%s: scan file %q: %w", op, cPath, err)
	}

	cfgs := make([]config.Config, len(sData))
	for i, d := range sData {
		b := config.BaseConfig{}
		if err := gurlf.Unmarshal(d, &b); err != nil {
			// <- log warn with string(d)
			return nil, fmt.Errorf("%s: item №[%d]: invalid base: %w",
				op, i, err)
		}
		b.ID = i
		cfg, err := handleType(&b, &d)
		if err != nil {
			return cfgs, fmt.Errorf("%s: cfg №[%d] failed: %w",
				op, i, err)
		}
		if err := resolveRepeat(i, &cfg, &cfgs); err != nil {
			return cfgs, fmt.Errorf("%s: cfg №[%d] repeat failed: %w",
				op, i, err)
		}

		cfgs[i] = cfg
	}

	return cfgs, nil
}

func handleType(b *config.BaseConfig, d *gscan.Data) (config.Config, error) {
	const op = "parser.handleType"

	var cfg config.Config
	tp := b.Type
	switch tp {
	case "http":
		cfg = &config.HTTPConfig{BaseConfig: *b}
	case "grpc":
		cfg = &config.GRPCConfig{BaseConfig: *b}
	case "repeat":
		cfg = &config.RepeatConfig{BaseConfig: *b}
	default:
		return nil, fmt.Errorf("%s: undefined cfg type: %q", op, tp)
	}

	if err := gurlf.Unmarshal(*d, cfg); err != nil {
		return nil, fmt.Errorf("%s: type %q: %w", op, tp, err)
	}
	return cfg, nil
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

	*cfg = (*cfgs)[r.TargetID].Clone()
	(*cfg).SetID(i)

	return nil
}
