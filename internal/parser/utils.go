package parser

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"unsafe"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func ParseHeaders(hdrs []byte, yield func([]byte, []byte)) {
	for len(hdrs) != 0 {
		kS := 0
		for kS < len(hdrs) && (isSpace(hdrs[kS]) || hdrs[kS] == '{') {
			kS++
		}

		kE := bytes.IndexByte(hdrs, ':')
		if kE == -1 {
			return
		}

		vS := kE + 1
		for vS < len(hdrs) && isSpace(hdrs[vS]) {
			vS++
		}
		vE := bytes.IndexByte(hdrs[vS:], '\n')
		if vE == -1 {
			vE = len(hdrs)
		} else {
			vE += vS
		}

		yield(hdrs[kS:kE], hdrs[vS:vE])
		hdrs = hdrs[vE:]
	}
}

func ParseContentType(ct *string) {
	s := *ct
	start, end := 0, len(s)

	for i := len(s) - 1; i > 0; i-- {
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

	if len(base) == 16 {
		for i := range len(base) {
			if (base[i] | 0x20) == ("application/json"[i] | 0x20) {
				*ct = "application/json"
				return
			}
		}
	}
	*ct = ""
}

func ParseBody(b []byte) []byte {
	lineStart := false
	readIdx, writeIdx := 0, 0
	for readIdx < len(b) && isSpace(b[readIdx]) {
		readIdx++
	}

	for readIdx < len(b) {
		ch := b[readIdx]

		if !lineStart {
			if ch == '\t' || ch == ' ' {
				readIdx++
				continue
			}
			lineStart = true
		}

		b[writeIdx] = b[readIdx]
		writeIdx++
		readIdx++

		if ch == '\n' {
			lineStart = false
		}
	}
	res := b[:writeIdx]
	for len(res) > 0 && isSpace(res[len(res)-1]) {
		res = res[:len(res)-1]
	}

	return res
}

func ParseResponse(res *[]byte, inst []byte) {
	const op = "parser.parseResponse"

	prefix := []byte("json:")
	jIdx := bytes.Index(inst, prefix)
	if jIdx == -1 {
		(*res) = nil
		return
	}

	kS := jIdx + len(prefix)
	for kS < len(inst) && isSpace(inst[kS]) {
		kS++
	}
	kE := len(inst)
	for kE > kS && (isSpace(inst[kE-1]) || inst[kE-1] == '}') {
		kE--
	}
	key := inst[kS:kE]
	pattern := append([]byte{'"'}, append(key, '"', ':')...)

	jsonStart := bytes.Index(*res, pattern)
	if jsonStart == -1 {
		(*res) = nil
		return
	}
	jsonStart += len(pattern)

	for jsonStart < len(*res) && isSpace((*res)[jsonStart]) {
		jsonStart++
	}
	if jsonStart >= len(*res) || (*res)[jsonStart] != '"' {
		(*res) = nil
		return
	}
	jsonStart++

	jsonEnd := jsonStart
	for jsonEnd < len(*res) {
		if (*res)[jsonEnd] == '"' && (*res)[jsonEnd-1] != '\\' {
			break
		}
		jsonEnd++
	}

	if jsonEnd >= len(*res) {
		(*res) = nil
		return
	}

	(*res) = (*res)[jsonStart:jsonEnd]
}

func ParseCookies(url *url.URL, cookies []*http.Cookie) []byte {
	const op = "parser.ParseCookies"

	if len(cookies) == 0 {
		return nil
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	buf.WriteByte('\n')
	buf.WriteByte('[')
	buf.WriteString(url.Host)
	buf.WriteByte(']')
	buf.WriteByte('\n')
	buf.WriteByte(' ')

	for _, c := range cookies {
		parts := strings.SplitSeq(c.Raw, ";")

		for p := range parts {
			if len(p) == 0 {
				continue
			}

			key, val, found := strings.Cut(p, "=")
			if !found {
				buf.WriteString(p)
				buf.WriteByte(':')
				buf.WriteByte('\n')
				continue
			}
			buf.WriteString(key)
			buf.WriteByte(':')
			buf.WriteString(val)

			buf.WriteByte('\n')
		}
	}

	buf.WriteByte('[')
	buf.WriteByte('\\')
	buf.WriteString(url.Host)
	buf.WriteByte(']')
	buf.WriteByte('\n')

	return buf.Bytes()
}

var skip = map[string]struct{}{
	"path":     {},
	"domain":   {},
	"expires":  {},
	"max-age":  {},
	"httponly": {},
	"secure":   {},
	"samesite": {},
}

func UnparseCookies(data []byte, yield func(string)) {
	const op = "parser.ParseLoadCookie"

	start := bytes.Index(data, []byte("]\n"))
	end := bytes.LastIndex(data, []byte("\n[\\"))
	if start == -1 || end == -1 || end <= start {
		return
	}
	data = data[start+2 : end]

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	for len(data) > 0 {
		var line []byte
		i := bytes.IndexByte(data, '\n')
		if i == -1 {
			line = data
			data = nil
		} else {
			line = data[:i]
			data = data[i+1:]
		}

		key, val, found := bytes.Cut(line, []byte(":"))
		key = bytes.TrimSpace(key)
		val = bytes.TrimSpace(val)

		if isMetadata(key) {
			continue
		}

		if !found || len(val) == 0 {
			buf.Write(key)
			buf.WriteByte(';')
			continue
		}

		buf.Write(key)
		buf.WriteByte('=')
		buf.Write(val)
		buf.WriteByte(';')
	}

	cks := buf.Bytes()
	cksStr := unsafe.String(unsafe.SliceData(cks), len(cks))

	yield(cksStr)
}

func isSpace(r byte) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f'
}

func isMetadata(k []byte) bool {
	switch len(k) {
	case 4:
		return equalFold(k, "path")
	case 6:
		return equalFold(k, "domain") || equalFold(k, "secure")
	case 7:
		return equalFold(k, "expires") || equalFold(k, "max-age")
	case 8:
		return equalFold(k, "httponly") || equalFold(k, "samesite")
	default:
		return false
	}
}

func equalFold(b []byte, lower string) bool {
	if len(b) != len(lower) {
		return false
	}
	for i := range b {
		if (b[i] | 0x20) != lower[i] {
			return false
		}
	}
	return true
}
