package parser

import (
	"testing"

	"github.com/Votline/Gurlf"
)

func TestFastExtract(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte("Type"), "http"},
		{[]byte("Body"), "hello"},
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

func TsetEqualFold(t *testing.T) {
	tests := []struct {
		input    []byte
		expected bool
	}{
		{[]byte("path"), true},
		{[]byte("Path"), true},
		{[]byte("PATH"), true},
		{[]byte("domain"), true},
		{[]byte("secure"), true},
		{[]byte("expires"), true},
		{[]byte("max-age"), true},
		{[]byte("httponly"), true},
		{[]byte("samesite"), true},
		{[]byte("nop"), false},
	}

	for i, tt := range tests {
		if equalFold(tt.input, "path") != tt.expected {
			t.Errorf("[%d]: expected %t, but got %t",
				i, tt.expected, equalFold(tt.input, "path"))
		}
	}
}

func BenchmarkEqualFold(b *testing.B) {
	for b.Loop() {
		equalFold([]byte("path"), "path")
	}
}
