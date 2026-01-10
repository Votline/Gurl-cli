package config

import (
	"gcli/internal/buffer"
	"unsafe"
)

type Config interface {
	Clone() Config
	Release()

	GetName() string
	SetName(string)

	GetID() int
	SetID(int)

	GetType() string
	SetType(string)
}

var (
	hItab uintptr
	hBuf  = buffer.NewRb[*HTTPConfig]()
	gBuf  = buffer.NewRb[*GRPCConfig]()
	gItab uintptr
)

func Init() {
	hBuf = buffer.NewRb[*HTTPConfig]()
	gBuf = buffer.NewRb[*GRPCConfig]()

	var hIface Config = &HTTPConfig{}
	hItab = *(*uintptr)(unsafe.Pointer(&hIface))

	var gIface Config = &GRPCConfig{}
	gItab = *(*uintptr)(unsafe.Pointer(&gIface))

	for i := 0; i < 10; i++ {
		hBuf.Write(&HTTPConfig{})
		gBuf.Write(&GRPCConfig{})
	}
}

type BaseConfig struct {
	Name string `gurlf:"config_name"`
	ID   int    `gurlf:"ID"`
	Type string `gurlf:"Type"`
	Resp string `gurlf:"Response"`
}

func defBase() *BaseConfig {
	return &BaseConfig{
		Name: "http_config",
		ID:   0,
		Type: "http",
		Resp: "",
	}
}

func (c *BaseConfig) Release()             {}
func (c *BaseConfig) Clone() Config        { cp := *c; return &cp }
func (c *BaseConfig) GetName() string      { return c.Name }
func (c *BaseConfig) SetName(nName string) { c.Name = nName }
func (c *BaseConfig) GetID() int           { return c.ID }
func (c *BaseConfig) SetID(nID int)        { c.ID = nID }
func (c *BaseConfig) GetType() string      { return c.Type }
func (c *BaseConfig) SetType(nType string) { c.Type = nType }

type HTTPConfig struct {
	BaseConfig
	Url     string `gurlf:"Url"`
	Method  string `gurlf:"Method"`
	Body    []byte `gurlf:"Body"`
	Headers []byte `gurlf:"Headers"`
}

func GetHTTP() (*HTTPConfig, uintptr) { return hBuf.Read(), hItab }
func (c *HTTPConfig) Release() {
	*c = HTTPConfig{}
	hBuf.Write(c)
}
func (c *HTTPConfig) Clone() Config {
	newCfg := hBuf.Read()
	*newCfg = *c
	return newCfg
}

type GRPCConfig struct {
	BaseConfig
}

func GetGRPC() (*GRPCConfig, uintptr) { return gBuf.Read(), gItab }
func (c *GRPCConfig) Release() { *c = GRPCConfig{}; gBuf.Write(c) }
func (c *GRPCConfig) Clone() Config {
	newCfg := gBuf.Read()
	*newCfg = *c
	return newCfg
}

type RepeatConfig struct {
	BaseConfig
	TargetID int `gurlf:"target_id"`
}
