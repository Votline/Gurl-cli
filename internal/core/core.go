package core

import (
	"gcli/internal/config"
	"go.uber.org/zap"
)

func handleConfig(cP, ckP string) error {
	return nil
}

func Start(cType, cPath, ckPath string, cCreate, ic bool, log *zap.Logger) error {
	if cCreate {
		return config.Create(cType, cPath)
	}
	return handleConfig(cPath, ckPath)
}
