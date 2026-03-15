package parser

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"
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

func TestParseRandom(t *testing.T) {
	tests := []struct {
		input []byte
	}{
		{[]byte("oneof=some,more,value")},
		{[]byte("oneof=uuid")},
		{[]byte("oneof=int")},
		{[]byte("oneof=int(1,10)")},
	}

	for i, tt := range tests {
		buf := make([]byte, 32)
		ParseRandom(tt.input, &buf)
		if len(buf) == 0 {
			t.Errorf("[%d]: expected len > 0, but got %d: %q", i, len(buf), string(buf))
		}
	}
}

func BenchmarkParseRandom(b *testing.B) {
	buf := make([]byte, 32)
	inst := []byte("")
	b.ResetTimer()
	for b.Loop() {
		ParseRandom(inst, &buf)
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct {
		input    []byte
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

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{123, "123"},
		{1234, "1234"},
		{12345, "12345"},
		{123456, "123456"},
		{1234567, "1234567"},
		{12345678, "12345678"},
		{123456789, "123456789"},
	}

	for i, tt := range tests {
		buf := make([]byte, 32)
		length := itoa(tt.input, &buf)
		buf = buf[:length]
		if string(buf) != tt.expected {
			t.Errorf("[%d]: expected %q, but got %q",
				i, tt.expected, string(buf))
		}
	}
}

func BenchmarkItoa(b *testing.B) {
	buf := make([]byte, 32)
	for b.Loop() {
		itoa(123456789, &buf)
	}
}
