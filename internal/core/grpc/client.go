package grpcClient

import (
	"log"

	"Gurl-cli/internal/config"
)

func InitConfig(path string) {
	config.SetupGRPC()
	log.Println(path)
}
