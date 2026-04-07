package parser

import (
	"bytes"
	"math/rand"
	"sync"
	"unsafe"

	gscan "github.com/Votline/Gurlf/pkg/scanner"
)

type chunker struct {
	data []byte
	done bool
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func fastExtract(data []byte, ents *[]gscan.Entry, need []byte) string {
	entries := *ents
	for _, ent := range entries {
		if bytes.Equal(data[ent.KeyStart:ent.KeyEnd], need) {
			vS, vE := ent.ValStart, ent.ValEnd
			tmp := data[vS:vE]
			tp := unsafe.String(unsafe.SliceData(tmp), len(tmp))
			return tp
		}
	}

	return ""
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

	var b [36]byte
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

func fastUUID(buf *[]byte) {
	if cap(*buf) < 36 {
		*buf = make([]byte, 36)
	} else {
		*buf = (*buf)[:36]
	}
	b := *buf

	u1 := rand.Uint64()
	u2 := rand.Uint64()

	// 8 chars
	for i := 0; i < 8; i++ {
		b[i] = hexChars[(u1>>(i*4))&0xf]
	}
	b[8] = '-'
	// 4 chars
	for i := 0; i < 4; i++ {
		b[9+i] = hexChars[(u1>>(32+i*4))&0xf]
	}
	b[13] = '-'
	// Version 4
	b[14] = '4'
	for i := 0; i < 3; i++ {
		b[15+i] = hexChars[(u1>>(48+i*4))&0xf]
	}
	b[18] = '-'
	// Variant
	b[19] = hexChars[(u2&0x3)+8]
	for i := 0; i < 3; i++ {
		b[20+i] = hexChars[(u2>>(2+i*4))&0xf]
	}
	b[23] = '-'
	// 12 chars
	for i := 0; i < 12; i++ {
		b[24+i] = hexChars[(u2>>(14+i*4))&0xf]
	}
}

func (c *chunker) next() ([]byte, bool) {
	if c.done {
		return nil, false
	}
	idx := bytes.IndexByte(c.data, ',')
	if idx == -1 {
		c.done = true
		return c.data, true
	}
	chunk := c.data[:idx]
	c.data = c.data[idx+1:]
	return chunk, true
}

func trimBytes(buf *[]byte, check func(byte) bool) {
	temp := *buf
	if check == nil {
		check = isSpace
	}

	for len(temp) > 0 && check(temp[0]) {
		temp = temp[1:]
	}
	for len(temp) > 0 && check(temp[len(temp)-1]) {
		temp = temp[:len(temp)-1]
	}

	*buf = temp
}
