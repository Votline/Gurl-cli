package parser

import (
	"bytes"
	"testing"
)

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
		buf := make([]byte, 36)
		ParseRandom(tt.input, &buf)
		if len(buf) == 0 {
			t.Errorf("[%d]: expected len > 0, but got %d: %q", i, len(buf), string(buf))
		}
	}
}

func BenchmarkParseRandom(b *testing.B) {
	buf := make([]byte, 36)
	inst := []byte("oneof=some,more")
	b.ResetTimer()
	for b.Loop() {
		ParseRandom(inst, &buf)
	}
}

func TestGetVarKey(t *testing.T) {
	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte("key=value"), []byte("value")},
		{[]byte("key=value,key2=value2"), []byte("value,key2=value2")},
		{[]byte("key=     value"), []byte("value")},
		{[]byte("    key     =     value"), []byte("value")},
		{[]byte("\n\t\tkey=\n\t\tvalue"), []byte("value")},
		{[]byte("no_equal"), nil},
		{[]byte("no_var_equal="), nil},
		{[]byte("="), nil},
		{[]byte("=}}}"), nil},
		{[]byte("key=}"), nil},
	}

	for i, tt := range tests {
		var key []byte
		GetVarKey(tt.input, &key)

		if !bytes.Equal(key, tt.expected) {
			t.Errorf("[%d]: expected %q, but got %q", i, tt.expected, key)
		}
	}
}

func BenchmarkGetVarKey(b *testing.B) {
	inst := []byte("key=value")
	b.ResetTimer()
	for b.Loop() {
		var key []byte
		GetVarKey(inst, &key)
	}
}

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expKey []byte
		expVal []byte
	}{
		{"Empty", []byte(""), nil, nil},
		{"NoEqual", []byte("no_equal"), nil, []byte("no_equal")},
		{"Standard", []byte("VAR=NAME VAL=VALUE"), []byte("NAME"), []byte("VALUE")},
		{"SpacesAndBraces", []byte("K=  KEY  V=  VAL  }}"), []byte("KEY"), []byte("VAL")},
		{"OnlyOneEqual", []byte("KEY="), []byte(""), []byte("KEY=")},
		{"MessyInput", []byte("=KEY=VAL=RES"), nil, []byte("=KEY=VAL=RES")},
		{"TrailingGarbage", []byte("K=K V=V extra"), []byte("K"), []byte("V extra")},
	}

	for i, tt := range tests {
		k, v := []byte{}, tt.input

		ParseEnv(&v, &k)
		if !bytes.Equal(k, tt.expKey) {
			t.Errorf("[%d]: expected key %q, but got %q", i, tt.expKey, k)
			continue
		}
		if !bytes.Equal(v, tt.expVal) {
			t.Errorf("[%d]: expected val %q, but got %q", i, tt.expVal, v)
		}
	}
}

func BenchmarkParseEnv(b *testing.B) {
	key, val := []byte{}, []byte("KEY=VALUE")
	for b.Loop() {
		ParseEnv(&val, &key)
	}
}

func TestSearchKey_MultiLineAndChaos(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		key      []byte
		expected []byte
	}{
		{
			name:     "Quoted multiline",
			data:     []byte("KEY=\"line1\nline2\nline3\""),
			key:      []byte("KEY"),
			expected: []byte("line1\nline2\nline3"),
		},
		{
			name:     "Unquoted stop at newline",
			data:     []byte("KEY=value_part1\nvalue_part2"),
			key:      []byte("KEY"),
			expected: []byte("value_part1"),
		},
		{
			name:     "Escaped quotes in multiline",
			data:     []byte("KEY=\"first line\nsecond \\\"quoted\\\" line\""),
			key:      []byte("KEY"),
			expected: []byte("first line\nsecond \\\"quoted\\\" line"),
		},

		{
			name:     "Key as suffix",
			data:     []byte("USER_ID=100\nID=200"),
			key:      []byte("ID"),
			expected: []byte("100"),
		},
		{
			name:     "Start equals end (empty quoted)",
			data:     []byte(`KEY=""`),
			key:      []byte("KEY"),
			expected: []byte(""),
		},
		{
			name:     "No newline at the end of file",
			data:     []byte("KEY=last_value_in_file"),
			key:      []byte("KEY"),
			expected: []byte("last_value_in_file"),
		},

		{
			name:     "Quote with backslash at the end",
			data:     []byte(`KEY="value with slash \\"`),
			key:      []byte("KEY"),
			expected: []byte(`value with slash \\`),
		},
		{
			name:     "Unclosed quote goes to EOF",
			data:     []byte(`KEY="no closing quote here`),
			key:      []byte("KEY"),
			expected: []byte("no closing quote here"),
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res []byte
			SearchKey(tt.data, tt.key, &res)
			if !bytes.Equal(res, tt.expected) {
				t.Errorf("[%d] \nData: %s\nKey:  %s\nGot:  %q\nExp:  %q", i,
					tt.data, tt.key, string(res), string(tt.expected))
			}
		})
	}
}

func BenchmarkSearchKey(b *testing.B) {
	data := []byte("KEY=value")
	b.ResetTimer()
	for b.Loop() {
		var key []byte
		SearchKey(data, []byte("KEY"), &key)
	}
}
