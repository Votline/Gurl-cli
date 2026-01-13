package config

import (
	"gcli/internal/buffer"
	"unsafe"
)

type Dependency struct {
	TargetID int
	Start    int
	End      int
	Key      string
}

type Config interface {
	Clone() Config
	Release()

	GetName() string
	SetName(string)

	GetID() int
	SetID(int)

	GetType() string
	SetType(string)

	RangeDeps(func (d Dependency))
	SetDependency(Dependency)
	Apply(int, int, string, []byte)
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
	ID    int    `gurlf:"ID"`
	Name  string `gurlf:"config_name"`
	Type  string `gurlf:"Type"`
	Resp  string `gurlf:"Response"`
	Deps         [6]Dependency
	ExtraDeps    []Dependency
	DepsLen      uint8
}

func defBase() *BaseConfig {
	return &BaseConfig{
		Type:  "http",
		Name:  "http_config",
	}
}

func (c *BaseConfig) Apply(int, int, string, []byte) {}
func (c *BaseConfig) Release()                       {}
func (c *BaseConfig) Clone() Config                  { cp := *c; return &cp }
func (c *BaseConfig) GetName() string                { return c.Name }
func (c *BaseConfig) SetName(nName string)           { c.Name = nName }
func (c *BaseConfig) GetID() int                     { return c.ID }
func (c *BaseConfig) SetID(nID int)                  { c.ID = nID }
func (c *BaseConfig) GetType() string                { return c.Type }
func (c *BaseConfig) SetType(nType string)           { c.Type = nType }
func (c *BaseConfig) RangeDeps(fn func(d Dependency)) {
	limit := c.DepsLen
	if limit > 6 { limit = 6 }

	for i := range limit {
		fn(c.Deps[i])
	}
	for _, d := range c.ExtraDeps {
		fn(d)
	}
}
func (c *BaseConfig) SetDependency(nDep Dependency)  {
	if c.DepsLen < 6 {
		c.Deps[c.DepsLen] = nDep
	} else {
		c.ExtraDeps = append(c.ExtraDeps, nDep)
	}
	c.DepsLen++
}

type HTTPConfig struct {
	BaseConfig
	Url     []byte `gurlf:"Url"`
	Method  []byte `gurlf:"Method"`
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
func (c *HTTPConfig) Apply(start, end int, key string, val []byte) {
	switch key {
	case "Url":
		c.Url = splice(c.Url, val, start, end)
	case "Method":
		c.Method = splice(c.Method, val, start, end)
	case "Body":
		c.Body = splice(c.Body, val, start, end)
	case "Headers":
		c.Headers = splice(c.Headers, val, start, end)
	}
}

type GRPCConfig struct {
	BaseConfig
}

func GetGRPC() (*GRPCConfig, uintptr) { return gBuf.Read(), gItab }
func (c *GRPCConfig) Release()        { *c = GRPCConfig{}; gBuf.Write(c) }
func (c *GRPCConfig) Clone() Config {
	newCfg := gBuf.Read()
	*newCfg = *c
	return newCfg
}
func (c *GRPCConfig) Apply(start, end int, key string, val []byte) {
	return
}

type RepeatConfig struct {
	BaseConfig
	TargetID int `gurlf:"target_id"`
}

func splice(orig, val []byte, start, end int) []byte {
	res := make([]byte, 0, len(orig)+len(val))
	res = append(res, orig[:start]...)
	res = append(res, val...)
	res = append(res, orig[end:]...)
	return res
}

