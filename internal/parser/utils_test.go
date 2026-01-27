package parser

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
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
		ParseResponse(&res, []byte(tt.input))
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
