package parser

import (
	"bytes"
	"strings"
)

func ParseHeaders(hdrs []byte, yield func([]byte, []byte)) {
	for len(hdrs) != 0 {
		kS := 0
		kE := bytes.IndexByte(hdrs, ':')
		if kE == -1 { return }

		vS := kE+1
		vE := len(hdrs)-1

		yield(hdrs[kS:kE], hdrs[vS:vE])
	}
}

func ParseContentType(ct *string) {
	s := *ct
	start, end := 0, len(s)

	for i := len(s)-1; i > 0; i-- {
		if s[i] == ';' {
			end = i
			break
		}
	}
	for start < end && isSpace(s[start]) {
		start++
	}

	for end > start && isSpace(s[end-1]) {
		end--
	}

	base := s[start:end]

	if len(base) == len("application/json") {
		if strings.EqualFold(base, "application/json") {
			*ct = "application/json"
			return
		}
	}
	*ct = ""
}

func isSpace(r byte) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f'
}
