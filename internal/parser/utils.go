package parser

import (
	"bytes"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
)

const (
	Error       = -1
	ExpectFail  = -2
	ExpectDone  = -3
	ExpectCrash = -4
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

func ParseWait(wait []byte) time.Duration {
	if len(wait) == 0 {
		return 0
	}

	t := wait[:len(wait)-1]
	d := atoi(t)
	if d == -1 {
		return Error
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

func ParseRandom(inst []byte, buf *[]byte) {
	if len(inst) == 0 {
		*buf = nil
		return
	}

	start := bytes.IndexByte(inst, '=')
	if start == -1 {
		*buf = nil
		return
	}
	start++

	randType := start
	for randType < len(inst) && !isSpace(inst[randType]) && inst[randType] != '(' && inst[randType] != '}' {
		randType++
	}

	haveComma := bytes.IndexByte(inst, ',')
	if haveComma != -1 && !bytes.Equal(inst[start:randType], []byte("int")) {
		args := bytes.Split(inst[start:], []byte(","))
		args[len(args)-1] = args[len(args)-1][:len(args[len(args)-1])-1] // remove '}'
		randIdx := rand.Intn(len(args))
		*buf = args[randIdx]
		return
	}

	if bytes.Equal(inst[start:randType], []byte("uuid")) {
		u, _ := uuid.New().MarshalText()
		*buf = u
		return
	} else if !bytes.Equal(inst[start:randType], []byte("int")) {
		*buf = nil
		return
	}

	startRange := bytes.IndexByte(inst[randType:], '(')
	if startRange == -1 {
		length := itoa(int(rand.Int63()), buf)
		*buf = (*buf)[:length]
		return
	}

	endRange := bytes.IndexByte(inst[randType:], ')')
	if endRange == -1 {
		*buf = nil
		return
	}

	startRange += randType + 1
	endRange += randType

	numsRange := inst[startRange:endRange]
	args := bytes.Split(numsRange, []byte(","))

	if len(args) != 2 {
		*buf = nil
		return
	}

	num1 := atoi(args[0])
	num2 := atoi(args[1])

	length := itoa(num1+rand.Intn(num2-num1+1), buf)
	*buf = (*buf)[:length]
}

func ParseExpect(expect []byte, resCode int) int {
	if len(expect) == 0 {
		return ExpectDone
	}

	end := bytes.IndexByte(expect, ';')
	if end == -1 {
		end = len(expect)
	}

	codes := bytes.Split(expect[:end], []byte(","))
	for _, code := range codes {
		codeInt := atoi(code)
		if codeInt == resCode {
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

func atoi(data []byte) int {
	res := 0
	foundDigit := false
	cur := 0
	for cur < len(data) && data[cur] >= '0' && data[cur] <= '9' {
		res = res*10 + int(data[cur]-'0')
		foundDigit = true
		cur++
	}

	if !foundDigit {
		return Error
	}

	return res
}

func itoa(n int, buf *[]byte) int {
	if n == 0 {
		(*buf)[0] = '0'
		return 1
	}

	var b [32]byte
	pos := len(*buf)

	for n > 0 && pos > 0 {
		pos--
		b[pos] = byte('0' + (n % 10))
		n /= 10
	}

	length := len(*buf) - pos
	copy((*buf)[:length], b[pos:])

	return length
}
