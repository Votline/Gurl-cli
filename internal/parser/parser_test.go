package parser

import (
	"bytes"
	"testing"

	"gcli/internal/config"

	"github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
	"go.uber.org/zap"
)

var raw = []byte(`
	[http_config]
	URL:http://localhost:8080
	Method:GET
	Body:hello
	Headers:Content-Type:application/json
	Type:http
	Response:hello
	[\http_config]`)

var repRaw = append(raw, []byte(`
		[rep]
		TargetID:0
		Type:repeat
		Response:something {RESPONSE id=0 json:token}
		[\rep]
`)...)

func yield(c config.Config) { c.Release() }

func TestParseStream(t *testing.T) {
	config.Init()
	log := zap.NewNop()

	d, _ := gurlf.Scan(raw)
	if err := ParseStream(&d, func(c config.Config) {
		if c.GetType() != "http" && c.GetType() != "repeat" {
			t.Errorf("expected %q, but got %q", "http or repeat", c.GetType())
		}
	}, log); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func BenchmarkParseStream(b *testing.B) {
	d, _ := gurlf.Scan(raw)
	log := zap.NewNop()

	for b.Loop() {
		if err := ParseStream(&d, yield, log); err != nil {
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

	tests := []struct {
		input *gscan.Data
	}{
		{&d[0]},
		{&d[1]},
	}
	instsPos := make([]instruction, 0, len(d))

	for _, tt := range tests {
		if err := handleInstructions(tt.input, insts, func(inst instruction) { instsPos = append(instsPos, inst) }); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkHandleInstructions(b *testing.B) {
	d, _ := gurlf.Scan(repRaw)

	instsPos := make([]instruction, 0, len(d))

	for b.Loop() {
		instsPos = instsPos[:0]
		if err := handleInstructions(&d[1], insts, func(inst instruction) { instsPos = append(instsPos, inst) }); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestHandleType(t *testing.T) {
	b := config.BaseConfig{
		Name: "http_config", ID: 15, Type: "http",
	}
	d, _ := gurlf.Scan(raw)

	config.Init()

	var cfg config.Config
	if err := handleType(&cfg, &b.Type, &d[0]); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := cfg.(*config.HTTPConfig); !ok {
		t.Errorf("Invalid type: %T", b)
	}

	cfg.Release()
}

func BenchmarkHandleType(b *testing.B) {
	d, _ := gurlf.Scan(raw)

	config.Init()
	var cfg config.Config
	for b.Loop() {
		base := config.BaseConfig{Name: "http_config", ID: 15, Type: "http"}
		if err := handleType(&cfg, &base.Type, &d[0]); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		cfg.Release()
	}
}

func TestApplyReplace(t *testing.T) {
	const op = "parser.TestApplyReplace"

	cfg := config.RepeatConfig{
		Replace: []byte(`
			[repa]
			URL:http://localhost:8080
			Body:hello
			[\repa]
		`),
		Orig: &config.HTTPConfig{
			URL: []byte("http://localhost:8443"),
			Body: []byte(`
				{
					"name": "Viz",
					"email": "yomi@duck.com",
					"password": "password"
				}
			`),
		},
	}

	if err := applyReplace(&cfg); err != nil {
		t.Fatalf("%s: %v", op, err)
	}

	orig := cfg.Orig.(*config.HTTPConfig)
	if !bytes.Equal(orig.URL, []byte("http://localhost:8080")) {
		t.Errorf("%s: expected %q, but got %q", op, orig.URL, []byte("http://localhost:8080"))
	}

	if !bytes.Equal(orig.Body, []byte("hello")) {
		t.Errorf("%s: expected %q, but got %q", op, orig.Body, []byte("hello"))
	}
}

func BenchmarkApplyReplace(b *testing.B) {
	cfg := config.RepeatConfig{
		Replace: []byte(`
			[repa]
			URL:http://localhost:8080
			Body:hello
			[\repa]
		`),
		Orig: &config.HTTPConfig{
			URL: []byte("http://localhost:8443"),
			Body: []byte(`
				{
					"name": "Viz",
					"email": "yomi@duck.com",
					"password": "password"
				}
			`),
		},
	}

	for b.Loop() {
		if err := applyReplace(&cfg); err != nil {
			b.Fatalf("%v", err)
		}
	}
}
