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
	type res struct {
		key []byte
		def []byte
	}
	tests := []struct {
		input    []byte
		expected res
	}{
		{[]byte("key=value"), res{[]byte("value"), nil}},
		{[]byte("key=value;def=value"), res{[]byte("value"), []byte("value")}},
		{[]byte("key=     value"), res{[]byte("value"), nil}},
		{[]byte("    key     =     value"), res{[]byte("value"), nil}},
		{[]byte("\n\t\tkey=\n\t\tvalue"), res{[]byte("value"), nil}},
		{[]byte("no_equal"), res{nil, nil}},
		{[]byte("no_var_equal="), res{nil, nil}},
		{[]byte("="), res{nil, nil}},
		{[]byte("=}}}"), res{nil, nil}},
		{[]byte("key=}"), res{nil, nil}},
	}

	var key []byte
	var def []byte
	for i, tt := range tests {
		GetVarKey(tt.input, &key, &def)

		if !bytes.Equal(key, tt.expected.key) {
			t.Errorf("[%d]: expected key %q, but got %q", i, tt.expected.key, key)
		}
		if !bytes.Equal(def, tt.expected.def) {
			t.Errorf("[%d]: expected def %q, but got %q", i, tt.expected.def, def)
		}
	}
}

func BenchmarkGetVarKey(b *testing.B) {
	inst := []byte("key=value ; def=value")
	var key []byte
	var def []byte

	b.ResetTimer()
	for b.Loop() {
		GetVarKey(inst, &key, &def)
	}
}

func TestParseEnv(t *testing.T) {
	type res struct {
		from []byte
		val  []byte
		def  []byte
	}

	tests := []struct {
		input []byte
		exp   res
	}{
		{[]byte(" key=value"), res{[]byte("value"), nil, nil}},
		{[]byte(" key=value ; from=os"), res{[]byte("value"), []byte("os"), nil}},
		{[]byte(" key=value ; from=os ; def=default"), res{[]byte("value"), []byte("os"), []byte("default")}},
		{[]byte(" key=     value"), res{[]byte("value"), nil, nil}},
		{[]byte(" key    =   value  "), res{[]byte("value"), nil, nil}},
		{[]byte(" key    =   value  ; from=os"), res{[]byte("value"), []byte("os"), nil}},
		{[]byte(""), res{nil, nil, nil}},
		{[]byte(" no_equal"), res{nil, nil, nil}},
		{[]byte(" key="), res{nil, nil, nil}},
		{[]byte(" =key=val;from==os"), res{[]byte("key=val"), []byte("=os"), nil}},
		{
			[]byte(" =more than one space ; = and here ; = here too"),
			res{[]byte("more than one space"), []byte("and here"), []byte("here too")},
		},
		{[]byte("  ==a lot == of == equal == "), res{[]byte("=a lot == of == equal =="), nil, nil}},
	}

	for i, tt := range tests {
		k, v, d := []byte{}, tt.input, []byte{}

		ParseEnv(&v, &k, &d)
		if !bytes.Equal(k, tt.exp.from) {
			t.Errorf("[%d]: expected key %q, but got %q", i, tt.exp.from, k)
			t.Errorf("[%d]: info: \nkey:%s\nval:%s\ndef:%s", i, k, v, d)
		}
		if !bytes.Equal(v, tt.exp.val) {
			t.Errorf("[%d]: expected val %q, but got %q", i, tt.exp.val, v)
			t.Errorf("[%d]: info: \nkey:%s\nval:%s\ndef:%s", i, k, v, d)
		}
		if !bytes.Equal(d, tt.exp.def) {
			t.Errorf("[%d]: expected def %q, but got %q", i, tt.exp.def, d)
			t.Errorf("[%d]: info: \nkey:%s\nval:%s\ndef:%s", i, k, v, d)
		}
	}
}

func BenchmarkParseEnv(b *testing.B) {
	key, val, def := []byte{}, []byte("KEY=VALUE"), []byte{}
	for b.Loop() {
		ParseEnv(&val, &key, &def)
	}
}

func TestSearchKey(t *testing.T) {
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
