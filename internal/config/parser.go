package config

import (
	"log"
	"sync"
	"time"
	"bytes"
	"errors"
	"strings"
	"strconv"
	"encoding/json"
)

func getNested(data interface{}, path string) (interface{}, bool) {
	keys := strings.Split(path, ".")
	var current interface{} = data
	for _, key := range keys {
		switch curr := current.(type) {
		case map[string]interface{}:
			val, ok := curr[key]
			if !ok {return nil, false}
			current = val
		case []interface{}:
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

func handleJson(source, inst string) ([]byte, error) {
	source = strings.Trim(source, `"`)
	source = strings.TrimSpace(source)

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(source), &data); err != nil {
		log.Printf("Error unmarshalling JSON from Response: %v\nSource: %v",
			err, source)
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
		log.Printf("Error marshalling response: %v", err)
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

func handleProcType(source, procType string) ([]byte, error) {
	if procType == "none" {
		return []byte(removeJsonShit(source)), nil
	}
	if strings.Contains(procType, "json:") {
		if !strings.HasPrefix(strings.TrimSpace(source), "{") {
			return handleString(source, procType)
		}
		return handleJson(source, procType)
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

func findID(data []byte, startIdx, endIdx int) (string, string, error) {
	template := string(data[startIdx : endIdx+startIdx+1])
	parts := strings.SplitN(template, " ", 3)
	if len(parts) < 3 {
		return "", "", errors.New("Invalid RESPONSE template")
	}

	idPart := strings.TrimPrefix(parts[1], "id=")
	_, err := strconv.Atoi(idPart)
	if err != nil {
		return "", "", errors.New("Invalid id in RESPONSE template")
	}

	procType := strings.Trim(parts[2], `}`)

	return idPart, procType, nil
}

func findIdx(data []byte) (startIdx, endIdx int) {
	startIdx = bytes.Index(data, []byte("RESPONSE id="))
	if startIdx == -1 {return -1, -1}
	endIdx = bytes.Index(data[startIdx:], []byte("}"))
	return
}

func parse[T Config](data []byte, cfgs []T) ([]byte, error) {
	startIdx, endIdx := findIdx(data)
	if startIdx == -1 || endIdx == -1 {
		return data, errors.New("Parse is not needed")
	}

	idPart, procType, err := findID(data, startIdx, endIdx)
	if err != nil {return nil, err}

	var sourceCfg T
	if !findSource(&sourceCfg, cfgs, idPart) {
		return nil, errors.New("Config not found. ID: " + idPart)
	}

	sourceResponse := sourceCfg.GetResponse()
	if sourceResponse == "" {
		return nil, errors.New("Config response is nil")
	}

	response, err := handleProcType(sourceResponse, procType)
	if err != nil {return nil, err}

	log.Printf("PARSER: Found template: %s", string(data[startIdx-1:startIdx+endIdx+1]))
	log.Printf("PARSER: Extracted value: %s", string(response))
	
	var result bytes.Buffer
	result.Write(data[:startIdx-1])
	result.Write(response)
	result.Write(data[startIdx+endIdx+1:])

	return result.Bytes(), nil
}

func Parsing[T Config](cfg T, cfgs []T) (T, error) {
	var zero T
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	wg.Add(1)
	go func(){
		defer wg.Done()
		for {
			url := cfg.GetUrl()
			newUrl, err := parse([]byte(url), cfgs)
			if err != nil {
				if err.Error() == "Parse is not needed" {
					break
				} else {errChan <- err; return}
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
				newHeaders, err := parse(headers, cfgs)
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
				newBody, err := parse(body, cfgs)
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

	wg.Wait()

	select{
	case err := <-errChan:
		log.Printf("An error was found during the operation of goroutin %v", err.Error())
		return zero, err
	default:
		return cfg, nil
	}

}
