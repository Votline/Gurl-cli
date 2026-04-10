// Package config create.go contains functions for create config.
package config

import (
	"fmt"
	"os"

	"github.com/Votline/Gurlf"
)

// Create accepts config type and path.
// It creates config file.
func Create(cType, cPath string) error {
	const op = "config.Create"

	var d []byte
	var err error
	switch cType {
	case "http":
		d, err = cHTTP()
	case "grpc":
		d, err = cGRPC()
	case "repeat":
		d, err = cRepeat()
	case "import":
		d, err = cImport()
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

// cHTTP returns raw data for HTTP config.
func cHTTP() ([]byte, error) {
	base := defBase()
	c := HTTPConfig{BaseConfig: *base}
	return gurlf.Marshal(c)
}

// cGRPC returns raw data for GRPC config.
func cGRPC() ([]byte, error) {
	base := defBase()
	base.Type = "grpc"
	base.Name = "grpc_config"
	c := GRPCConfig{BaseConfig: *base}
	return gurlf.Marshal(&c)
}

// cRepeat returns raw data for Repeat config.
func cRepeat() ([]byte, error) {
	base := defBase()
	base.Type = "repeat"
	base.Name = "repeat_config"
	c := &RepeatConfig{BaseConfig: *base}
	return gurlf.Marshal(c)
}

// cImport returns raw data for Import config.
func cImport() ([]byte, error) {
	base := defBase()
	base.Type = "import"
	base.Name = "import_config"
	c := &ImportConfig{BaseConfig: *base}
	return gurlf.Marshal(c)
}

// cMix returns raw data for HTTP and GRPC config.
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
