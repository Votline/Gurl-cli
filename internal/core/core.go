package core

import (
	"log"
)

func HandleFlags(cfg, cfgType, cfgPath string, cfgCreate bool) {
	if cfg != "" {
		log.Println(cfg)
		return
	}
	if !cfgCreate {
		log.Fatalln("Write --config-create")
	}

	InitConfig(cfgPath, cfgType)
}
