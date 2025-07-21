package httpClient

import (
	"log"

	"Gurl-cli/internal/config"
)

func InitConfig(path string) {
	config.SetupHTTP()
	log.Println(path)
}
