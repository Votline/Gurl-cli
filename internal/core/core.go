package core

import (
	"log"
	
	"Gurl-cli/internal/core/http"
	"Gurl-cli/internal/core/grpc"
)

func HandleFlags(cfg, cfgType, cfgPath string, cfgCreate bool) {
	if cfg != "" {
		log.Println(cfg)
		return
	}
	if !cfgCreate {
		log.Fatalln("Write --config-create")
	}

	switch cfgType {
	case "http":
		httpClient.InitConfig(cfgPath)
	case "grpc":
		grpcClient.InitConfig(cfgPath)
	}
}
