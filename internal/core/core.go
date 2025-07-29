package core

import (
	"log"

	"Gurl-cli/internal/config"
	"Gurl-cli/internal/transport"
	gen "Gurl-cli/internal/generate"
)

func prettyPrint(data interface{}, indent string) {
	switch v := data.(type) {
	case map[string]interface{}:
		log.Println(indent+"[")
		for key, val := range v {
			switch valTyped := val.(type) {
			case map[string]interface{}, []interface{}:
				log.Printf("%s    %s:", indent, key)
				prettyPrint(valTyped, indent + "    ")
			default:
				log.Printf("%s    %s: %v", indent, key, valTyped)
			}
		}
		log.Println(indent+"]")
	case []interface{}:
		for _, elem := range v {
			prettyPrint(elem, indent+"    ")
		}
	default:
		log.Printf("%s%v", indent, v)
	}
}

func handleCfgType(rawCfg interface{}) {
	switch cfg := rawCfg.(type) {
	case *config.HTTPConfig:
		handleHTTP(cfg)
	default:
		log.Fatalf("Invalid config type: %v", cfg)
	}
}

func HandleFlags(cfgType, cfgPath string, cfgCreate bool) {
	if !cfgCreate {
		handleRequest(cfgPath)
		return
	}
	gen.InitConfig(cfgPath, cfgType)
}

func handleRequest(cfgPath string) {
	rawCfg, err := config.Decode(cfgPath)
	if err != nil {
		log.Fatalf("Error when trying to get the config:\n%v", err.Error())
	}

	if cfgs, ok := rawCfg.([]interface{}); ok {
		for _, cfg := range cfgs {
			handleCfgType(cfg)
		}
		return
	}
	handleCfgType(rawCfg)
}

func handleHTTP(cfg *config.HTTPConfig) {
	var err error
	var res transport.Result
	switch cfg.Method {
	case "GET":
		res, err = transport.Get(cfg)
	case "POST":
		res, err = transport.Post(cfg)
	case "PUT":
		res, err = transport.Put(cfg)
	case "DELETE":
		res, err = transport.Del(cfg)
	}
	if err != nil {
		log.Fatalf("Error when trying to make a %v request:\n%v", cfg.Method, err.Error())
	}

	log.Println(res.Raw.Status)
	if res.JSON != nil {
		prettyPrint(res.JSON, "    ")
	} else if res.RawBody != nil {
		log.Println(string(res.RawBody))
	} else {
		log.Println("Empty response body")
	}
}
