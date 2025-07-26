package core

import (
	"log"

	"Gurl-cli/internal/config"
	"Gurl-cli/internal/transport"
	gen "Gurl-cli/internal/generate"
)

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

	switch cfg := rawCfg.(type) {
	case *config.HTTPConfig:
		handleHTTP(cfg)
	default:
		log.Fatalln("Invalid config type")
	}
}

func handleHTTP(cfg *config.HTTPConfig) {
	var err error
	var res transport.Result
	switch cfg.Method {
	case "GET":
		res, err = transport.Get(cfg.Url)
	case "POST":
		res, err = transport.Post(cfg)
	}
	if err != nil {
		log.Fatalf("Error when trying to make a %v request:\n%v", cfg.Method, err.Error())
	}

	log.Println(res.Raw.Status)
	if res.JSON != nil {
		for key, value := range res.JSON {
			log.Printf("%s: %v", key, value)
		}
	} else if res.RawBody != nil {
		log.Println(string(res.RawBody))
	} else {
		log.Println("Empty response body")
	}
}
