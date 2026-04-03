package parser

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unsafe"

	gscan "github.com/Votline/Gurlf/pkg/scanner"
)

const (
	Error       = -1
	ExpectFail  = -2
	ExpectDone  = -3
	ExpectCrash = -4
	WS          = -5
	WSwhile     = -6
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

	prefix := []byte("json:")
	jIdx := bytes.Index(inst, prefix)
	if jIdx == -1 {
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

func ParseWait(wait []byte) time.Duration {
	if len(wait) == 0 {
		return 0
	}

	t := wait[:len(wait)-1]
	d := atoi(t)
	if d == -1 {
		return Error
	}

	if wait[len(wait)-2] == 'm' && wait[len(wait)-1] == 's' {
		return time.Duration(d) * time.Millisecond
	}

	switch wait[len(wait)-1] {
	case 's':
		return time.Duration(d) * time.Second
	case 'm':
		return time.Duration(d) * time.Minute
	case 'h':
		return time.Duration(d) * time.Hour
	default:
		return Error
	}
}

const hexChars = "0123456789abcdef"

func ParseExpect(expect []byte, resCode int) int {
	if len(expect) == 0 {
		return ExpectDone
	}

	end := bytes.IndexByte(expect, ';')
	if end == -1 {
		end = len(expect)
	}

	ch := chunker{data: expect[:end], done: false}
	for {
		chunk, ok := ch.next()
		if !ok {
			break
		}
		if len(chunk) == 0 {
			return Error
		}
		if atoi(chunk) == resCode {
			return ExpectDone
		}
	}

	if end == len(expect) {
		return ExpectFail
	}

	end++ // skip ';'
	for end < len(expect) && isSpace(expect[end]) {
		end++
	}
	if end == len(expect) {
		return ExpectFail
	}

	separator := bytes.IndexByte(expect[end:], '=')
	if separator == -1 {
		return ExpectFail
	}
	separator += end + 1 // skip '='

	for separator < len(expect) && isSpace(expect[separator]) {
		separator++
	}
	if separator == len(expect) {
		return ExpectFail
	}

	action := expect[separator:]

	if bytes.Equal(action, []byte("crash")) {
		return ExpectCrash
	}

	id := atoi(action)
	return id
}

func ApplyVars(vars []gscan.Data, varsMap map[string][]byte) {
	parseWithMap(vars, func(key string, val []byte, n string) {
		if len(key) == 0 {
			return
		}
		varsMap[key] = val
	})
}

func ApplyEnvs(envs []gscan.Data) {
	type entry struct {
		key string
		val []byte
	}

	fileGroup := make(map[string][]entry)
	parseWithMap(envs, func(key string, val []byte, name string) {
		if len(key) == 0 {
			return
		}

		if _, err := os.Stat(name); os.IsNotExist(err) {
			os.Setenv(key, unsafe.String(unsafe.SliceData(val), len(val)))
			return
		}

		fileGroup[name] = append(fileGroup[name], entry{key, val})
	})

	for name, entries := range fileGroup {
		existingContent, _ := os.ReadFile(name)

		for {
			endIdx := bytes.IndexByte(existingContent, '\n')
			if endIdx == -1 {
				break
			}
			divIdx := bytes.IndexByte(existingContent, '=')
			if divIdx == -1 {
				break
			}

			key := unsafe.String(unsafe.SliceData(existingContent[:divIdx]), len(existingContent[:divIdx]))
			val := existingContent[divIdx+1:]

			found := false
			for _, ent := range entries {
				if ent.key == key {
					found = true
					break
				}
			}
			// if key is not found in entries, it means that it's a new key
			if !found {
				entries = append(entries, entry{key, val})
			}

			existingContent = existingContent[endIdx+1:]
		}

		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			continue
		}
		defer f.Close()

		for _, ent := range entries {
			f.WriteString(ent.key)
			f.WriteString("=")
			f.Write(ent.val)
			f.WriteString("\n")
		}
	}
}

func parseWithMap(data []gscan.Data, yield func(string, []byte, string)) {
	for _, v := range data {
		for _, ent := range v.Entries {
			if ent.ValEnd == 0 {
				continue
			}
			kS := ent.KeyStart
			for kS < len(v.RawData) && isSpace(v.RawData[kS]) {
				kS++
			}
			kE := ent.KeyEnd
			for kE > kS && (isSpace(v.RawData[kE-1]) || v.RawData[kE-1] == '}') {
				kE--
			}

			vS := ent.ValStart
			for vS < len(v.RawData) && isSpace(v.RawData[vS]) {
				vS++
			}
			vE := ent.ValEnd
			for vE > vS && (isSpace(v.RawData[vE-1]) || v.RawData[vE-1] == '}') {
				vE--
			}

			key := unsafe.String(unsafe.SliceData(v.RawData[kS:kE]), kE-kS)
			val := v.RawData[vS:vE]
			name := unsafe.String(unsafe.SliceData(v.Name), len(v.Name))
			yield(key, val, name)
		}
	}
}

func DetectWS(u *[]byte) int {
	url := *u
	end := bytes.Index(url, []byte("://"))
	if end == -1 {
		return Error
	}

	scheme := url[:end]
	trimSpaceBytes(&scheme)

	if bytes.Equal(scheme, []byte("ws")) {
		return WS
	}

	sep := bytes.Index(scheme, []byte(":"))
	if sep == -1 {
		return Error
	}

	wsType := scheme[:sep]
	trimSpaceBytes(&wsType)

	if bytes.Equal(wsType, []byte("while")) && bytes.Equal(scheme[sep+1:], []byte("ws")) {
		*u = url[sep+1:]
		return WSwhile
	}

	return Error
}
