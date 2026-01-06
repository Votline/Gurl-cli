package config

import (
	"fmt"
	"os"

	"github.com/Votline/Gurlf"
)

func Create(cType, cPath string) error {
	const op = "config.Create"

	var d []byte
	var err error
	switch cType {
	case "http":
		d, err = cHttp()
	case "grpc":
		d, err = cGrpc()
	case "repeat":
		d, err = cRepeat()
	case "mixed":
		d, err = cMix()
	default:
		return fmt.Errorf("%s: undefined config", op)
	}
	if err != nil {
		return fmt.Errorf("%s: create cfg: %w", op, err)
	}

	f, err := os.Create(cPath)
	if err != nil {
		return fmt.Errorf("%s: create file (path=%q): %w",
			op, cPath, err)
	}
	defer f.Close()

	return gurlf.Encode(f, d)
}

func cHttp() ([]byte, error) {
	base := defBase()
	c := HTTPConfig{BaseConfig: *base}
	return gurlf.Marshal(c)
}
func cGrpc() ([]byte, error) {
	base := defBase()
	base.Type = "grpc"
	base.Name = "grpc_config"
	c := GRPCConfig{BaseConfig: *base}
	return gurlf.Marshal(&c)
}
func cRepeat() ([]byte, error) {
	base := defBase()
	base.Type = "repeat"
	base.Name = "repeat_config"
	c := RepeatConfig{BaseConfig: *base}
	return gurlf.Marshal(&c)
}
func cMix() ([]byte, error) {
	base := defBase()
	httpCfg := HTTPConfig{BaseConfig: *base}
	base.Name = "grpc_config"
	grpcCfg := GRPCConfig{BaseConfig: *base}
	grpcCfg.ID = 1

	hData, err := gurlf.Marshal(httpCfg)
	if err != nil {
		return nil, err
	}
	gData, err := gurlf.Marshal(grpcCfg)
	if err != nil {
		return nil, err
	}

	return append(hData, gData...), nil
}
