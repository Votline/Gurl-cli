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

func handleHTTP(cfg *config.HTTPConfig, cfgPath string, ic bool) {
	var err error
	var res transport.Result
	switch cfg.Method {
	case "GET":
		res, err = transport.Get(cfg, ic)
	case "POST":
		res, err = transport.Post(cfg, ic)
	case "PUT":
		res, err = transport.Put(cfg, ic)
	case "DELETE":
		res, err = transport.Del(cfg, ic)
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

func handleGRPC(cfg *config.GRPCConfig, cfgPath string) {
	res, err := transport.GRPC(cfg)
	if err != nil {
		log.Fatalf("Error when trying to make a gRPC request:\n%v", err)
	}

	if res.JSON != nil {
		prettyPrint(res.JSON, "    ")
	} else if res.RawBody != nil {
		log.Println(string(res.RawBody))
	} else {
		log.Println("Empty resonse body")
	}

	cfg.SetResponse(string(res.RawBody))
	config.ConfigUpd(cfg, cfgPath)
}

func handleRequest(cfgPath string, ic bool) {
	cfgs, err := config.Decode(cfgPath)
	if err != nil {
		log.Fatalf("Error when trying to get the config:\n%v", err.Error())
	}

	for _, c := range cfgs {
		cfg, err := config.Parsing(c, cfgs)
		if err != nil {
			log.Printf("Parse error: %v", err)
			continue
		}
		switch v := cfg.(type) {
		case *config.HTTPConfig:
			handleHTTP(v, cfgPath, ic)
		case *config.GRPCConfig:
			handleGRPC(v, cfgPath)
		default:
			log.Printf("Invalid config type %v", v)
		}
	}
}

func HandleFlags(cfgType, cfgPath string, cfgCreate, ic bool) {
	if !cfgCreate {
		switch cfgType {
		case "http", "grpc":
			handleRequest(cfgPath, ic)
			return
		default:
			log.Fatalf("Invalid config type: %s", cfgType)
		}
	}
	if err := gen.InitConfig(cfgPath, cfgType); err != nil {
		log.Fatalf("Generate config error: %v", err)
	}
}
