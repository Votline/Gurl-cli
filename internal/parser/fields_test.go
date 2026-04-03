package parser

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"
	"unsafe"

	gurlf "github.com/Votline/Gurlf"
	gscan "github.com/Votline/Gurlf/pkg/scanner"
)

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		input   string
		expKeys []string
		expVals []string
	}{
		{"Host: google.com", []string{"Host"}, []string{"google.com"}},
		{
			"User-Agent: Mozilla/5.0 (Win)",
			[]string{"User-Agent"},
			[]string{"Mozilla/5.0 (Win)"},
		},
		{
			"\n\t\tContent-Type: application/xml\n",
			[]string{"Content-Type"},
			[]string{"application/xml"},
		},
		{
			"Accept: text/html\nContent-Type: application/json",
			[]string{"Accept", "Content-Type"},
			[]string{"text/html", "application/json"},
		},
		{
			`{
		    Content-Type: application/json,
		    Authorization: Bearer token
			}`,
			[]string{"Content-Type", "Authorization"},
			[]string{"application/json,", "Bearer token"},
		},
	}

	for i, tt := range tests {
		ParseHeaders([]byte(tt.input), func(b1, b2 []byte) {
			if !slices.Contains(tt.expKeys, string(b1)) {
				t.Errorf("[%d]: expected keys %q, but got %q",
					i, tt.expKeys, string(b1))
			}
			if !slices.Contains(tt.expVals, string(b2)) {
				t.Errorf("[%d]: expected vals %q, but got %q",
					i, tt.expVals, string(b2))
			}
		})
	}
}

func BenchmarkParseHeaders(b *testing.B) {
	raw := []byte("Content-Type: application/json")

	for b.Loop() {
		ParseHeaders(raw, func(b1, b2 []byte) {})
	}
}

func TestParseContentType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{" application/json\n\n", "application/json"},
		{"application/xml", ""},
		{"\n\tapplication/json\n\t", "application/json"},
	}

	for i, tt := range tests {
		ParseContentType(&tt.input)
		if tt.input != tt.expected {
			t.Errorf("[%d]: expected %q, but got %q",
				i, tt.expected, tt.input)
		}
	}
}

func BenchmarkParseContentType(b *testing.B) {
	for b.Loop() {
		str := "application/json"
		ParseContentType(&str)
	}
}

func TestParseBody(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\n\t\tpretty\n\t\tbody\n", "pretty\nbody"},
		{"pretty\nbody", "pretty\nbody"},
		{"one string body", "one string body"},
		{"pret\n\tty\n\t\tbo\ndy\n", "pret\nty\nbo\ndy"},
	}

	for i, tt := range tests {
		res := ParseBody([]byte(tt.input))
		if string(res) != tt.expected {
			t.Errorf("[%d]: expected %q, but got %q",
				i, tt.expected, string(res))
		}
	}
}

func BenchmarkParseBody(b *testing.B) {
	for b.Loop() {
		ParseBody([]byte("\n\t\tpretty\n\t\tbody\n"))
	}
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		input    string
		inst     string
		expected string
	}{
		{`"token":   "fjhklghdfsdiuflg"`, `{RESPONSE id=0 json:token}`, `fjhklghdfsdiuflg`},
		{`"\nToken": "fj\nhklghdfsd\tiuflg\r"`, `{RESPONSE id=15 json:\nToken}`, `fj\nhklghdfsd\tiuflg\r`},
	}

	for i, tt := range tests {
		res := []byte(tt.input)
		ParseResponse(&res, []byte(tt.inst))
		if string(res) != tt.expected {
			t.Errorf("[%d]: expected %q, but got %q",
				i, tt.expected, string(res))
		}
	}
}

func BenchmarkParseResponse(b *testing.B) {
	var res []byte
	for b.Loop() {
		ParseResponse(&res, []byte(`"json": "token"`))
	}
}

func TestParseCookies(t *testing.T) {
	tests := []struct {
		input    *http.Cookie
		expected string
	}{
		{&http.Cookie{Raw: "a=b"}, "\n[localhost.com]\n a:b\n[\\localhost.com]\n"},
		{&http.Cookie{Raw: "a=b; c=d"}, "\n[localhost.com]\n a:b\n c:d\n[\\localhost.com]\n"},
		{
			&http.Cookie{Raw: "Domain=google.com; Path=/; Expires=Wed, 09 Jun 2023 10:18:14 GMT; HttpOnly; Secure; SameSite=None; a=b"},
			"\n[localhost.com]\n Domain:google.com\n Path:/\n Expires:Wed, 09 Jun 2023 10:18:14 GMT\n HttpOnly:\n Secure:\n SameSite:None\n a:b\n[\\localhost.com]\n",
		},
	}

	for i, tt := range tests {
		res := ParseCookies(&url.URL{Scheme: "http", Host: "localhost.com"}, []*http.Cookie{tt.input})
		if string(res) != tt.expected {
			t.Errorf("[%d]: expected %q, but got %q",
				i, tt.expected, string(res))
		}
	}
}

func BenchmarkParseCookies(b *testing.B) {
	for b.Loop() {
		ParseCookies(&url.URL{Scheme: "https", Host: "google.com"}, []*http.Cookie{{Raw: "a=b"}})
	}
}

func TestUnparseCookies(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\n[localhost.com]\n a:b\n[\\localhost.com]\n", "a=b;"},
		{"\n[localhost.com]\n a:b\n c:d\n[\\localhost.com]\n", "a=b;c=d;"},
		{"\n[localhost.com]\n a:b\n c:d\n dOmaIN:htp\n Path:/\n[\\localhost.com]\n", "a=b;c=d;"},
	}

	for i, tt := range tests {
		var res string
		UnparseCookies([]byte(tt.input), func(ck string) {
			res = ck
		})
		if len(res) != len(tt.expected) {
			t.Errorf("[%d]: expected len: %d, but got %d\n got: %q",
				i, len(tt.expected), len(res), tt.expected)
		}

		if res != tt.expected {
			t.Errorf("[%d]: expected %q, but got %q",
				i, tt.expected, res)
		}
	}
}

func BenchmarkUnparseCookies(b *testing.B) {
	for b.Loop() {
		UnparseCookies([]byte("\n[localhost.com]\n a:b\n[\\localhost.com]\n"), func(ck string) {})
	}
}

func TestParseWait(t *testing.T) {
	tests := []struct {
		input    []byte
		expected time.Duration
	}{
		{[]byte("10s"), 10 * time.Second},
		{[]byte("15m"), 15 * time.Minute},
		{[]byte("02h"), 2 * time.Hour},
		{[]byte("1"), -1},
		{[]byte("1s"), 1 * time.Second},
		{[]byte("15ms"), 15 * time.Millisecond},
	}

	for i, tt := range tests {
		dur := ParseWait(tt.input)
		if dur != tt.expected {
			t.Errorf("[%d]: expected %d, but got %d", i, tt.expected, dur)
		}
	}
}

func BenchmarkParseWait(b *testing.B) {
	for b.Loop() {
		ParseWait([]byte("10s"))
	}
}

func TestParseExpect(t *testing.T) {
	tests := []struct {
		input    []byte
		resCode  int
		expected int
	}{
		{[]byte("200"), 200, ExpectDone},
		{[]byte("200,,201"), 201, Error},
		{[]byte("some"), 200, ExpectFail},
		{[]byte("200,404"), 200, ExpectDone},
		{[]byte("404"), 200, ExpectFail},
		{[]byte("200;fail=crash"), 200, ExpectDone},
		{[]byte("404;fail=crash"), 200, ExpectCrash},
		{[]byte("404;fail=15"), 200, 15},
		{[]byte("some;fail=1"), 200, 1},
	}

	for i, tt := range tests {
		res := ParseExpect(tt.input, tt.resCode)
		if res != tt.expected {
			t.Errorf("[%d]: expected %d, but got %d", i, tt.expected, res)
		}
	}
}

func BenchmarkParseExpect(b *testing.B) {
	for b.Loop() {
		ParseExpect([]byte("200"), 200)
	}
}

func TestParseWithMap(t *testing.T) {
	type result struct {
		key  string
		val  string
		name string
	}

	rawData := []byte("SetVariables:`\n[vars]\nStandartName:`\n    multi-\n   line`\n[\\vars]\n`")
	rawScan, _ := gurlf.Scan(rawData)

	tests := []struct {
		name     string
		input    []gscan.Data
		expected []result
	}{
		{
			name: "Basic trim and brace removal",
			input: []gscan.Data{
				{
					Name:    []byte("config.json"),
					RawData: []byte("  {KEY_1} = {VALUE_1}  "),
					Entries: []gscan.Entry{
						{KeyStart: 3, KeyEnd: 9, ValStart: 13, ValEnd: 21},
					},
				},
			},
			expected: []result{
				{key: "KEY_1", val: "VALUE_1", name: "config.json"},
			},
		},
		{
			name: "Multiple entries and spaces",
			input: []gscan.Data{
				{
					Name:    []byte("env"),
					RawData: []byte("  MY_KEY  =  MY_VAL  "),
					Entries: []gscan.Entry{
						{KeyStart: 0, KeyEnd: 10, ValStart: 13, ValEnd: 21},
					},
				},
			},
			expected: []result{
				{key: "MY_KEY", val: "MY_VAL", name: "env"},
			},
		},
		{
			name:  "Multiple entries and spaces",
			input: rawScan,
			expected: []result{
				{key: "StandardName", val: "multi-line", name: "vars"},
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseWithMap(tt.input, func(key string, val []byte, name string) {
				if key == "" || val == nil || name == "" {
					t.Errorf("[%d]: got empty key or val or name", i)
				}
				valStr := unsafe.String(unsafe.SliceData(val), len(val))
				if key != tt.expected[0].key {
					t.Errorf("[%d]: expected key %q, but got %q", i, tt.expected[0].key, key)
				}
				if valStr != tt.expected[0].val {
					t.Errorf("[%d]: expected val %q, but got %q", i, tt.expected[0].val, valStr)
				}
				if name != tt.expected[0].name {
					t.Errorf("[%d]: expected name %q, but got %q", i, tt.expected[0].name, name)
				}
			})
		})
	}
}

func BenchmarkParseWithMap(b *testing.B) {
	data := []gscan.Data{
		{
			Name:    []byte("config.json"),
			RawData: []byte("  {KEY_1} = {VALUE_1}  "),
			Entries: []gscan.Entry{
				{KeyStart: 2, KeyEnd: 9, ValStart: 12, ValEnd: 21},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		parseWithMap(data, func(key string, val []byte, name string) {})
	}
}

func TestDetectWS(t *testing.T) {
	tests := []struct {
		input       []byte
		expected    int
		expectedURL []byte
	}{
		{[]byte("ws://localhost:8080/ws"), WS, []byte("ws://localhost:8080/ws")},
		{[]byte("ws://localhost:8080/ws/test"), WS, []byte("ws://localhost:8080/ws/test")},
		{[]byte("while:ws://localhost:8080/ws"), WSwhile, []byte("ws://localhost:8080/ws")},
		{[]byte("http://localhost:8080/ws"), Error, []byte("http://localhost:8080/ws")},
	}

	for i, tt := range tests {
		res := DetectWS(&tt.input)
		if res != tt.expected {
			t.Errorf("[%d]: expected %d, but got %d", i, tt.expected, res)
		}
		if !slices.Equal(tt.expectedURL, tt.input) {
			t.Errorf("[%d]: expected %q, but got %q", i, tt.expectedURL, tt.input)
		}
	}
}

func BenchmarkDetectWS(b *testing.B) {
	url := []byte("ws://localhost:8080/ws")
	for b.Loop() {
		DetectWS(&url)
	}
}
