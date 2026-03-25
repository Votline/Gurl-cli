package parser

import (
	"bytes"
	"math/rand"
	"unsafe"

	gscan "github.com/Votline/Gurlf/pkg/scanner"
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

func ParseVars(vars []gscan.Data, varsMap map[string][]byte) {
	for _, v := range vars {
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
			varsMap[key] = val
		}
	}
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

func ParseEnvLine(line []byte, key []byte, val *[]byte) {
	if len(line) == 0 {
		return
	}

	idx := bytes.Index(line, key)
	if idx == -1 {
		return
	}
	idx += len(key)

	for idx < len(line) && isSpace(line[idx]) {
		idx++
	}
	if idx == len(line) {
		return
	}

	start := bytes.IndexByte(line[idx:], '=')
	if start == -1 {
		return
	}
	start += idx + 1 // skip '='

	for start < len(line) && (isSpace(line[start]) || line[start] == '"') {
		start++
	}

	end := len(line)
	for end > start && (isSpace(line[end-1]) || line[end-1] == '"') {
		end--
	}

	*val = line[start:end]
}
