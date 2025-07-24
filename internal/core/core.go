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
	}
	gen.InitConfig(cfgPath, cfgType)
}

func handleRequest(cfgPath string) {
	cfg, err := config.Decode(cfgPath)
	if err != nil {
		log.Fatalf("Error when trying to get the config:\n%v", err.Error())
	}

	switch v := cfg.(type) {
	case *config.HTTPConfig:
		switch v.Method {
		case "GET":
			res, err := transport.Get(v.Url);
			if err != nil {
				log.Fatalf("Error when trying to make a GET request:\n%v", err.Error())
			}
			log.Println(res)
		default:
			log.Fatalf("Invalid method type: %v\nValid ones: GET,POST,PUT,DELETE", v.Method)
		}
	}
}
