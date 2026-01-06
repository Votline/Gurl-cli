package parser

import (
	"gcli/internal/config"
	"testing"

	"github.com/Votline/Gurlf"
)

var raw = []byte(`
	[http_config]
	id:15
	type:http
	resp:hello
	[\http_config]`)

func TestParseData(t *testing.T) {
	d, _ := gurlf.Scan(raw)

	cfgs, err := parseData(&d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfgs) != 1 {
		t.Errorf("expected len 1 but got %d", len(cfgs))
	}
}
func BenchmarkParseData(b *testing.B) {
	d, _ := gurlf.Scan(raw)

	for b.Loop() {
		_, err := parseData(&d)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestHandleType(t *testing.T) {
	var b config.BaseConfig = config.BaseConfig{
		Name: "http_config", ID: 15, Type: "http"}
	d, _ := gurlf.Scan(raw)

	var cfg config.Config
	if err := handleType(&cfg, &b.Type, &d[0]); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := cfg.(*config.HTTPConfig); !ok {
		t.Errorf("Invalid type: %T", b)
	}
}
func BenchmarkHandleType(b *testing.B) {
	d, _ := gurlf.Scan(raw)

	var cfg config.Config
	for b.Loop() {
		var base config.BaseConfig = config.BaseConfig{Name: "http_config", ID: 15, Type: "http"}
		if err := handleType(&cfg, &base.Type, &d[0]); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestResolveRepeat(t *testing.T) {
	repCfg := config.RepeatConfig{
		BaseConfig: config.BaseConfig{
			Name: "repeat_config", ID: 15, Type: "repeat"},
		TargetID: 0,
	}
	var cfg config.Config = &repCfg
	cfgs := []config.Config{
		&config.HTTPConfig{
			BaseConfig: config.BaseConfig{
				Name: "http_config",ID: 0, Type: "repeat"} },
		cfg,
	}
	var i int = 1
	
	if err := resolveRepeat(i, &cfg, &cfgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
func BenchmarkResolveRepeat(b *testing.B) {
	repCfg := config.RepeatConfig{
		BaseConfig: config.BaseConfig{
			Name: "repeat_config", ID: 15, Type: "repeat"},
		TargetID: 0,
	}
	var cfg config.Config = &repCfg
	cfgs := []config.Config{
		&config.HTTPConfig{
			BaseConfig: config.BaseConfig{
				Name: "http_config",ID: 0, Type: "repeat"} },
		cfg,
	}
	var i int = 1
	
	for b.Loop() {
		if err := resolveRepeat(i, &cfg, &cfgs); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestFastExtractType(t *testing.T) {
	d, _ := gurlf.Scan(raw)
	
	if tp := fastExtractType(&d[0].RawData, &d[0].Entries); tp != "http" {
		t.Errorf("expected %q, but got %q", "http", tp)
	}
}
func BenchmarkFastExtractType(b *testing.B) {
	d, _ := gurlf.Scan(raw)
	
	for b.Loop() {
		fastExtractType(&d[0].RawData, &d[0].Entries)
	}
}
