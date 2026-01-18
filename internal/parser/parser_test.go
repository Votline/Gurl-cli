package parser

import (
	"gcli/internal/config"
	"testing"

	"github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
)

var raw = []byte(`
	[http_config]
	ID:0
	Type:http
	Response:hello
	[\http_config]`)
var repRaw = append(raw, []byte(`
		[rep]
		Target_ID:0
		Type:repeat
		Response:something {RESPONSE id=0 json:token}
		[\rep]
`)...)

func yield(c config.Config) { c.Release() }

func TestParseStream(t *testing.T) {
	config.Init()
	d, _ := gurlf.Scan(raw)
	if err := ParseStream(&d, func(c config.Config) {
		if c.GetType() != "http" && c.GetType() != "repeat" {
			t.Errorf("expected %q, but got %q", "http or repeat", c.GetType())
		}
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
func BenchmarkParseStream(b *testing.B) {
	d, _ := gurlf.Scan(raw)

	for b.Loop() {
		if err := ParseStream(&d, yield); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestHandleRepeat(t *testing.T) {
	d, _ := gurlf.Scan(repRaw)

	tests := []struct {
		input    *gscan.Data
		expected int
	}{
		{&d[0], -1},
		{&d[1], 0},
	}

	for i, tt := range tests {
		tID, err := handleRepeat(tt.input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tID != tt.expected {
			t.Errorf("[%d]: expected %d, but got %d", i, tt.expected, tID)
		}
	}
}
func BenchmarkHandleRepeat(b *testing.B) {
	d, _ := gurlf.Scan(repRaw)

	for b.Loop() {
		if _, err := handleRepeat(&d[0]); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestHandleInstructions(t *testing.T) {
	d, _ := gurlf.Scan(repRaw)
	
	tests := []struct{
		input   *gscan.Data
		expected int
	}{
		{&d[0], -1},
		{&d[1], 0},
	}
	insts := [][]byte{[]byte("RESPONSE id=")}
	instsPos := make([]instruction, 0, len(d))
	
	for i, tt := range tests {
		tID, err := handleInstructions(tt.input, &insts, func(inst instruction) { instsPos = append(instsPos, inst) })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tID != tt.expected {
			t.Errorf("[%d]: expected %d, but got %d", i, tt.expected, tID)
		}
	}
}
func BenchmarkHandleInstructions(b *testing.B) {
	d, _ := gurlf.Scan(repRaw)
	
	insts := [][]byte{[]byte("RESPONSE id=")}
	instsPos := make([]instruction, 0, len(d))

	for b.Loop() {
		instsPos = instsPos[:0]
		if _, err := handleInstructions(&d[1], &insts, func(inst instruction) { instsPos = append(instsPos,  inst)}); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestHandleType(t *testing.T) {
	var b config.BaseConfig = config.BaseConfig{
		Name: "http_config", ID: 15, Type: "http"}
	d, _ := gurlf.Scan(raw)

	config.Init()

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

	config.Init()
	var cfg config.Config
	for b.Loop() {
		var base config.BaseConfig = config.BaseConfig{Name: "http_config", ID: 15, Type: "http"}
		if err := handleType(&cfg, &base.Type, &d[0]); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		cfg.Release()
	}
}

func TestFastExtract(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte("Type"), "http"},
		{[]byte("ID"), "0"},
		{[]byte("Response"), "hello"},
	}
	d, _ := gurlf.Scan(raw)

	for i, tt := range tests {
		if tp := fastExtract(d[0].RawData, &d[0].Entries, tt.input); tp != tt.expected {
			t.Errorf("[%d]: expected %q, but got %q", i, tt.expected, tp)
		}
	}
}
func BenchmarkFastExtract(b *testing.B) {
	d, _ := gurlf.Scan(raw)

	for b.Loop() {
		fastExtract(d[0].RawData, &d[0].Entries, []byte("type"))
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct{
		input []byte
		expected int
	}{
		{[]byte("10"), 10},
		{[]byte("15 "), 15},
		{[]byte("02"), 2},
	}

	for i, tt := range tests {
		tID := atoi(tt.input)
		if tID != tt.expected {
			t.Errorf("[%d]: expected %d, but got %d", i, tt.expected, tID)
		}
	}
}
func BenchmarkAtoi(b *testing.B) {
	for b.Loop() {
		atoi([]byte("2"))
	}
}
