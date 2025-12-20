package config

import (
	"sync"
	"maps"
	"time"
	"bytes"
	"errors"
	"strings"
	"strconv"
	"encoding/json"

	"go.uber.org/zap"
)

var instructions = map[string]int{
	"RESPONSE": 3,
	"COOKIES": 2,
}

func getNested(data any, path string) (any, bool) {
	keys := strings.Split(path, ".")
	var current any = data
	for _, key := range keys {
		switch curr := current.(type) {
		case map[string]any:
			val, ok := curr[key]
			if !ok {return nil, false}
			current = val
		case []any:
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(curr) {
				return nil, false
			}
			current = curr[idx]
		default:
			return nil, false
		}
	}
	return current, true
}

func (p *Parser) handleJson(source, inst string) ([]byte, error) {
	source = strings.Trim(source, `"`)
	source = strings.TrimSpace(source)

	var data map[string]any
	if err := json.Unmarshal([]byte(source), &data); err != nil {
		p.log.Error("Unmarshalling JSON error",
			zap.String("source", source),
			zap.Error(err))
		return nil, err
	}

	parts := strings.SplitN(inst, ":", 2)
	field := parts[1]
	value, exists := getNested(data, field)
	if !exists {
		return nil, errors.New("Field not found in response")
	}

	strValue, ok := value.(string)
	if ok {
		strValue = strings.ReplaceAll(strValue, `"`, `'`)
		return []byte(strValue), nil
	}
	
	res, err := json.Marshal(value)
	if err != nil {
		p.log.Error("Marshalling response error", zap.Error(err))
		return nil, err
	}
	return res, nil
}

func handleString(source, inst string) ([]byte, error) {
	parts := strings.SplitN(inst, ":", 2)
	if len(parts) < 2 {
		return nil, errors.New("Invalid instruction")
	}
	return []byte(source), nil
}

func (p *Parser) handleProcType(source, procType string) ([]byte, error) {
	if procType == "none" {
		return []byte(removeJsonShit(source)), nil
	}
	if strings.Contains(procType, "json:") {
		if !strings.HasPrefix(strings.TrimSpace(source), "{") {
			return handleString(source, procType)
		}
		return p.handleJson(source, procType)
	}
	return []byte(source), nil
}

func removeJsonShit(s string) string {
	s = strings.NewReplacer(
		`"`, "",
		`\`, "",
		`{`, "",
		`}`, "",
	).Replace(s)
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func findSource[T Config] (sourceCfg *T, cfgs []T, id string) bool {
	for _, cfg := range cfgs {
		if cfg.GetID() == id {
			*sourceCfg = cfg
			return true
		}
	}
	return false
}

func templateData(data []byte, startIdx, endIdx int) (instType, idPart, procType string, err error) {
	template := string(data[startIdx : endIdx+startIdx])
	parts := strings.Split(template, " ")

	instType = parts[0]
	minParts := instructions[instType]

	if len(parts) < minParts {
		err = errors.New("Invalid template")
		return
	}

	idPart = strings.TrimPrefix(parts[1], "id=")
	if _, convErr := strconv.Atoi(idPart); convErr != nil {
		err = errors.New("Invalid id: " + idPart)
		return
	}

	if minParts == 3 {
		procType = parts[2]
	}

	return
}

func findIdx(data []byte) (startIdx, endIdx int) {
	endIdx = -1
	startIdx = -1
	
	for inst := range instructions {
		if idx := bytes.Index(data, []byte(inst+" id=")); idx != -1 {
			startIdx = idx
			if breakIdx := bytes.Index(data[startIdx:], []byte("}")); breakIdx != - 1 {
				endIdx = breakIdx
				return
			}
		}
	}
	return
}

func (p *Parser) parse(data []byte, cfgs []Config) ([]byte, bool, Config, error) {
	var zero Config

	startIdx, endIdx := findIdx(data)
	if startIdx == -1 {
		return data, false, zero, errors.New("Parse is not needed")
	}

	instType, idPart, procType, err := templateData(data, startIdx, endIdx)
	if err != nil {return nil, false, zero, err}

	var sourceCfg Config
	if !findSource(&sourceCfg, cfgs, idPart) {
		return nil, false, zero, errors.New("Config not found. ID: " + idPart)
	}

	if instType == "REPEAT" {
		return nil, true, sourceCfg, nil
	}

	sourceResponse := sourceCfg.GetResponse()
	if sourceResponse == "" {
		return nil, false, zero, errors.New("Config response is nil")
	}

	newData, err := p.handleProcType(sourceResponse, procType)
	if err != nil {return nil, false, zero, err}

	p.log.Debug("PARSER: Replaced template",
		zap.String("template", string(data[startIdx-1:startIdx+endIdx+1])),
		zap.String("extracted value", string(newData)))
	
	var result bytes.Buffer
	result.Write(data[:startIdx-1])
	result.Write(newData)
	result.Write(data[startIdx+endIdx+1:])

	return result.Bytes(), false, zero, nil
}

func applyReplace[T Config](baseCfg T, repeatedCfg T) T {
	replacements := repeatedCfg.GetReplace()
	if len(replacements) == 0 {
		return baseCfg
	}

	body := baseCfg.GetBody()
	if len(body) > 0 {
		var bodyMap map[string]any
		if err := json.Unmarshal(body, &bodyMap); err == nil {
			maps.Copy(bodyMap, replacements)
			newBody, _ := json.Marshal(bodyMap)
			baseCfg.SetBody(newBody)
		}
	}

	headersRaw, _ := baseCfg.GetHeaders()
	if len(headersRaw) > 0 {
		var headers map[string]any
		if err := json.Unmarshal(headersRaw, &headers); err == nil {
			for k, v := range replacements {
				if _, ok := headers[k]; ok {
					headers[k] = v
				}
			}
			newHeaders, _ := json.Marshal(headers)
			baseCfg.SetHeaders(newHeaders)
		}
	}

	if val, ok := replacements["url"]; ok {
		if s, ok := val.(string); ok {
			baseCfg.SetUrl(s)
		}
	}

	return baseCfg
}

func (p *Parser) Parsing(cfg Config, cfgs []Config) (Config, error) {
	var zero Config
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	if cfg.GetType() == "repeated" {
		id, err := strconv.Atoi(cfg.GetID())
		if err != nil {return zero, err}
		baseCfg := cfgs[id-1]
		finalCfg := applyReplace(baseCfg, cfg)
		return p.Parsing(finalCfg, cfgs)
	}

	wg.Add(1)
	go func(){
		defer wg.Done()
		for {
			url := cfg.GetUrl()
			newUrl, repeat, newCfg, err := p.parse([]byte(url), cfgs)
			if err != nil {
				if err.Error() == "Parse is not needed" {
					break
				} else {errChan <- err; return}
			} else if repeat {
				cfg = newCfg
				continue
			}
			cfg.SetUrl(string(newUrl))
		}
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for {
			headers, err := cfg.GetHeaders()
			if err != nil {errChan <- err; return}
			if headers != nil {
				newHeaders, _, _, err := p.parse(headers, cfgs)
				if err != nil {
					if err.Error() == "Parse is not needed" {
						break
					} else {errChan <- err; return}
				}
				if newHeaders != nil {
					cfg.SetHeaders(newHeaders)
				}
			} else {break}
			time.Sleep(100*time.Millisecond)
		}
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for {
			body := cfg.GetBody()
			if body != nil {
				newBody, _, _, err := p.parse(body, cfgs)
				if err != nil {
					if err.Error() == "Parse is not needed" {
						break
					} else {errChan <- err; return}
				}
				if newBody != nil {
					cfg.SetBody(newBody)
				}
			} else {break}
			time.Sleep(100*time.Millisecond)
		}
	}()
/*
	wg.Add(1)
	go func(){
		defer wg.Done()
	}()
*/
	wg.Wait()

	select{
	case err := <-errChan:
		p.log.Error("Error in goroutines", zap.Error(err))
		return zero, err
	default:
		return cfg, nil
	}
}
