package parser

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"unsafe"
)

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

	tpStart := bytes.IndexByte(inst, ':')
	if tpStart == -1 {
		return
	}

	tpEnd := len(inst)
	for tpEnd > tpStart && isSpace(inst[tpEnd-1]) {
		tpEnd--
	}

	jsonStart := bytes.IndexByte(*res, ':')
	if jsonStart == -1 {
		return
	}

	jsonStart++
	for jsonStart < len(*res) && isSpace((*res)[jsonStart]) {
		jsonStart++
	}

	jsonEnd := jsonStart + 1
	for jsonEnd < len(*res) {
		if (*res)[jsonEnd] == '"' && (*res)[jsonEnd-1] != '\\' {
			break
		}
		jsonEnd++
	}

	(*res) = (*res)[jsonStart+1 : jsonEnd]
}

var ckBuf bytes.Buffer

func ParseCookies(url *url.URL, cookies []*http.Cookie) []byte {
	const op = "parser.ParseCookies"

	if len(cookies) == 0 {
		return nil
	}

	ckBuf.Reset()
	ckBuf.WriteByte('\n')
	ckBuf.WriteByte('[')
	ckBuf.WriteString(url.Host)
	ckBuf.WriteByte(']')
	ckBuf.WriteByte('\n')
	ckBuf.WriteByte(' ')

	for _, c := range cookies {
		parts := strings.SplitSeq(c.Raw, ";")

		for p := range parts {
			if len(p) == 0 {
				continue
			}

			key, val, found := strings.Cut(p, "=")
			if !found {
				ckBuf.WriteString(p)
				ckBuf.WriteByte(':')
				ckBuf.WriteByte('\n')
				continue
			}
			ckBuf.WriteString(key)
			ckBuf.WriteByte(':')
			ckBuf.WriteString(val)

			ckBuf.WriteByte('\n')
		}
	}

	ckBuf.WriteByte('[')
	ckBuf.WriteByte('\\')
	ckBuf.WriteString(url.Host)
	ckBuf.WriteByte(']')
	ckBuf.WriteByte('\n')

	return ckBuf.Bytes()
}

var unpBuf bytes.Buffer

func UnparseCookies(data []byte) []*http.Cookie {
	const op = "parser.ParseLoadCookie"

	unpBuf.Reset()

	start := bytes.Index(data, []byte("]\n"))
	end := bytes.LastIndex(data, []byte("\n[\\"))
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	data = data[start+3 : end]

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
		if !found || len(val) == 0 {
			unpBuf.Write(key)
			unpBuf.WriteByte(';')
			continue
		}

		unpBuf.Write(key)
		unpBuf.WriteByte('=')
		unpBuf.Write(val)
		unpBuf.WriteByte(';')
	}

	raw := unpBuf.Bytes()
	ckStr := unsafe.String(unsafe.SliceData(raw), len(raw))
	header := http.Header{}
	header.Set("Cookie", ckStr)
	cookies := (&http.Request{Header: header}).Cookies()

	return cookies
}

func isSpace(r byte) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f'
}
