package grpcClient

import (
	"log"
	"encoding/json"

	"Gurl-cli/internal/config"
)

func InitConfig(path string) {
	json, err := json.MarshalIndent(config.SetupGRPC(), "", "    ")
	if err != nil {log.Fatalln(err)}
	log.Println(string(json))
}
