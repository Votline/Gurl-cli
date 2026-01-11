package parser

import (
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
			[]string{"User-Agent"}, []string{"Mozilla/5.0 (Win)"}},
		{
			"Content-Type: application/xml",
			[]string{"Content-Type"}, []string{"application/xml"}},
		{
			"Accept: text/html\nContent-Type: application/json",
			[]string{"Accept", "Content-Type"},
			[]string{"text/html", "application/json"},
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
		ParseHeaders(raw, func(b1, b2 []byte) { return })
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
