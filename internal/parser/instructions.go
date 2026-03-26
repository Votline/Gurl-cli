package parser

import (
	"bytes"
	"math/rand"
)

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
		data := inst[start:]
		if len(data) > 0 && data[len(data)-1] == '}' {
			data = data[:len(data)-1]
		}

		count := bytes.Count(data, []byte(",")) + 1
		randIdx := rand.Intn(count)

		ch := chunker{data: data}
		var chunk []byte

		for i := 0; i <= randIdx; i++ {
			chunk, _ = ch.next()
		}
		*buf = chunk
		return
	}

	if bytes.Equal(inst[start:randType], []byte("uuid")) {
		fastUUID(buf)
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
	ch := chunker{data: numsRange}

	arg1, ok1 := ch.next()
	arg2, ok2 := ch.next()
	if !ok1 || !ok2 || len(arg1) == 0 || len(arg2) == 0 {
		*buf = nil
		return
	}

	num1 := atoi(arg1)
	num2 := atoi(arg2)

	length := itoa(num1+rand.Intn(num2-num1+1), buf)
	*buf = (*buf)[:length]
}

func GetVarKey(inst []byte, key *[]byte) {
	if len(inst) == 0 {
		*key = nil
		return
	}

	start := bytes.IndexByte(inst, '=')
	if start == -1 {
		*key = nil
		return
	}
	start++

	for start < len(inst) && isSpace(inst[start]) {
		start++
	}

	end := len(inst)
	for end > start && (isSpace(inst[end-1]) || inst[end-1] == '}') {
		end--
	}

	*key = inst[start:end]
}

func ParseEnv(val *[]byte, key *[]byte) {
	inst := *val

	if len(inst) == 0 {
		*key = nil
		return
	}

	start := bytes.IndexByte(inst, '=')
	if start == -1 {
		*key = nil
		return
	}
	start++ // skip '='

	for start < len(inst) && isSpace(inst[start]) {
		start++
	}

	end := bytes.IndexByte(inst[start:], ' ')

	for end > start && isSpace(inst[end-1]) {
		end--
	}
	end += start

	*key = inst[start:end]

	inst = inst[end:]
	start = bytes.IndexByte(inst, '=')
	if start == -1 {
		*val = nil
		return
	}
	start++ // skip '='

	for start < len(inst) && isSpace(inst[start]) {
		start++
	}

	end = len(inst)
	for end > start && (isSpace(inst[end-1]) || inst[end-1] == '}') {
		end--
	}

	*val = inst[start:end]
}

func SearchKey(data, key []byte, val *[]byte) {
	if len(data) == 0 {
		*val = nil
		return
	}

	idx := bytes.Index(data, key)
	if idx == -1 {
		*val = nil
		return
	}
	idx += len(key)

	for idx < len(data) && isSpace(data[idx]) {
		idx++
	}
	if idx == len(data) {
		*val = nil
		return
	}

	start := bytes.IndexByte(data[idx:], '=')
	if start == -1 {
		*val = nil
		return
	}
	start += idx + 1 // skip '='

	var end int

	if data[start] == '"' {
		start++ // skip '"'

		seacrhFrom := start
		for {
			relIdx := bytes.IndexByte(data[seacrhFrom:], '"')
			if relIdx == -1 {
				end = len(data)
				break
			}

			absIdx := seacrhFrom + relIdx

			if absIdx > start && data[absIdx-1] == '\\' {
				seacrhFrom = absIdx + 1
				continue
			}

			end = absIdx
			break
		}
	} else {
		relEnd := bytes.IndexByte(data[start:], '\n')
		if relEnd == -1 {
			end = len(data)
		} else {
			end = start + relEnd
		}
	}

	for end > start && isSpace(data[end-1]) {
		end--
	}

	*val = data[start:end]
}
