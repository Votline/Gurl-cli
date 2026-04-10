// Package parser utils.go contains helper functions for parser.
package parser

import (
	"bytes"
	"math/rand"
	"sync"

	gscan "github.com/Votline/Gurlf/pkg/scanner"
)

// chunker struct is a simple iterator
// Used in ParseRandom, ParseExpect and others
type chunker struct {
	data []byte
	done bool
}

// bufPool is a sync.Pool for bytes.Buffer
// Used in ParseCookies and UnparseCookies.
var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// fastExtract extracts data from config data by key.
func fastExtract(data []byte, ents *[]gscan.Entry, need []byte) string {
	entries := *ents
	for _, ent := range entries {
		if bytes.Equal(data[ent.KeyStart:ent.KeyEnd], need) {
			vS, vE := ent.ValStart, ent.ValEnd
			tmp := data[vS:vE]
			tp := string(tmp)
			return tp
		}
	}

	return ""
}

// isSpace is a helper function for detect space or tab
func isSpace(r byte) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f'
}

// isMetadata is a helper function for detect metadata
// Used in UnparseCookies.
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

// equalFold compares case bytes with string.
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

// atoi converts []byte to int.
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

// itoa converts int to []byte.
// Returns length of result.
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

// fastUUID generates UUID v4 in a fast way.
// Accepts buffer to write UUID.
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
	for i := range 8 {
		b[i] = hexChars[(u1>>(i*4))&0xf]
	}
	b[8] = '-'
	// 4 chars
	for i := range 4 {
		b[9+i] = hexChars[(u1>>(32+i*4))&0xf]
	}
	b[13] = '-'
	// Version 4
	b[14] = '4'
	for i := range 3 {
		b[15+i] = hexChars[(u1>>(48+i*4))&0xf]
	}
	b[18] = '-'
	// Variant
	b[19] = hexChars[(u2&0x3)+8]
	for i := range 3 {
		b[20+i] = hexChars[(u2>>(2+i*4))&0xf]
	}
	b[23] = '-'
	// 12 chars
	for i := range 12 {
		b[24+i] = hexChars[(u2>>(14+i*4))&0xf]
	}
}

// next returns next chunk from data.
// If data is empty, returns empty slice and true.
// If data is not empty, returns slice with next chunk and false.
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

// trimBytes trims bytes from buffer.
// Accepts buffer and check function.
// If check function is nil, used 'isSpace'.
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
