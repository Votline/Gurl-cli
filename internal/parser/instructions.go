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

func GetVarKey(inst []byte, key, def *[]byte) {
	if len(inst) == 0 {
		*key = nil
		*def = nil
		return
	}

	eqIdx := bytes.IndexByte(inst, '=')
	if eqIdx == -1 {
		*key = nil
		*def = nil
		return
	}
	eqIdx++ // skip '='

	content := inst[eqIdx:]

	for {
		sepIdx := bytes.IndexByte(content, ';')
		if sepIdx != -1 && sepIdx-1 > 0 && content[sepIdx-1] != '\\' {
			*key = content[:sepIdx]

			sepIdx++ // skip ';'
			eqIdx = bytes.IndexByte(content[sepIdx:], '=')
			if eqIdx == -1 {
				*def = nil
				break
			}
			eqIdx++ // skip '='

			*def = content[sepIdx+eqIdx:]

			trimBytes(def, func(b byte) bool {
				return isSpace(b) || b == '}'
			})
			break
		} else if sepIdx == -1 {
			*key = content
			*def = nil
			break
		}
	}

	trimBytes(key, func(b byte) bool {
		return isSpace(b) || b == '}'
	})
}

func ParseEnv(from, key, def *[]byte) {
	if len((*from)) == 0 {
		*key = nil
		return
	}

	inst := *from
	if inst[0] == '{' {
		inst = inst[1:]
	}
	if inst[len(inst)-1] == '}' {
		inst = inst[:len(inst)-1]
	}
	// Trim only one brace, so they can be used in value

	skipPrefix := bytes.IndexByte(inst, ' ')
	if skipPrefix == -1 {
		*key = nil
		return
	}
	inst = inst[skipPrefix:]

	nextVal := func(buf *[]byte, source *[]byte) {
		current := *source
		start := bytes.IndexByte(current, '=')
		if start == -1 {
			*buf = nil
			return
		}
		start++ // skip '='

		end := bytes.IndexByte(current[start:], ';')
		if end == -1 {
			*buf = current[start:]
			*source = nil
		} else {
			end += start
			*buf = current[start:end]
			*source = current[end+1:]
		}

		trimBytes(buf, func(b byte) bool {
			return isSpace(b)
		})
	}

	nextVal(key, &inst)
	nextVal(from, &inst)
	nextVal(def, &inst)

	/*
	   start := bytes.IndexByte((*val), '=')

	   	if start == -1 {
	   		*key = nil
	   		return
	   	}

	   start++ // skip '='

	   	for start < len((*val)) && isSpace((*val)[start]) {
	   		start++
	   	}

	   end := bytes.IndexByte((*val)[start:], ' ')

	   	for end > start && isSpace((*val)[end-1]) {
	   		end--
	   	}

	   end += start

	   	if start == end || end < start || end > len((*val)) {
	   		*key = nil
	   		return
	   	}

	   *key = (*val)[start:end]

	   (*val) = (*val)[end:]
	   start = bytes.IndexByte((*val), '=')

	   	if start == -1 {
	   		*val = nil
	   		return
	   	}

	   start++ // skip '='

	   	for start < len((*val)) && isSpace((*val)[start]) {
	   		start++
	   	}

	   end = len((*val))

	   	for end > start && (isSpace((*val)[end-1]) || (*val)[end-1] == '}') {
	   		end--
	   	}

	   	if start == end || end < start || end > len((*val)) {
	   		*key = nil
	   		return
	   	}

	   *val = (*val)[start:end]
	*/
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

			if absIdx > start {
				if absIdx-1 >= 0 && data[absIdx-1] == '\\' {
					if absIdx-2 >= 0 && data[absIdx-2] != '\\' {
						seacrhFrom = absIdx + 1
						continue
					}
				}
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
