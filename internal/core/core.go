package core

import (
	"log"

	"Gurl-cli/internal/config"
	"Gurl-cli/internal/transport"
	gen "Gurl-cli/internal/generate"
)

func prettyPrint[T any](data T, indent string) {
	switch v := any(data).(type) {
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

func handleHTTP(cfg *config.HTTPConfig, cfgPath string) {
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

	cfg.SetResponse(string(res.RawBody))
	config.ConfigUpd(cfg, cfgPath)
}

func handleHTTPRequest(cfgPath string) {
	cfgs, err := config.Decode[*config.HTTPConfig](cfgPath)
	if err != nil {
		log.Fatalf("Error when trying to get the config:\n%v", err.Error())
	}

	for _, cfg := range cfgs {
		log.Println(string(cfg.GetBody()))
		parsed, err := config.Parsing(cfg, cfgs)
		if err != nil {
			log.Printf("Parse error: %v", err)
			continue
		}
		log.Println(string(parsed.GetBody()))
		if err != nil {
			log.Println(err)
			handleHTTP(parsed, cfgPath)
		}
	}
}

func HandleFlags(cfgType, cfgPath string, cfgCreate bool) {
	if !cfgCreate {
		switch cfgType {
		case "http":
			handleHTTPRequest(cfgPath)
			return
		default:
		return
		}
	}
	gen.InitConfig(cfgPath, cfgType)
}
