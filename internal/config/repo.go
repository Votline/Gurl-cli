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

	GetStart() int
	SetStart(int)

	GetType() string
	SetType(string)

	GetResp() string
	SetResp(string)

	SetOrig(Config)
	UnwrapExec() Config
	RangeDeps(func(d Dependency))
	SetDependency(Dependency)
	Apply(int, int, string, []byte)
}

var (
	hBuf  = buffer.NewRb[*HTTPConfig]()
	gBuf  = buffer.NewRb[*GRPCConfig]()
	rBuf  = buffer.NewRb[*RepeatConfig]()
	hItab uintptr
	gItab uintptr
	rItab uintptr
)

func Init() {
	hBuf = buffer.NewRb[*HTTPConfig]()
	gBuf = buffer.NewRb[*GRPCConfig]()
	rBuf = buffer.NewRb[*RepeatConfig]()

	var hIface Config = &HTTPConfig{}
	hItab = *(*uintptr)(unsafe.Pointer(&hIface))

	var gIface Config = &GRPCConfig{}
	gItab = *(*uintptr)(unsafe.Pointer(&gIface))

	var rIface Config = &RepeatConfig{}
	rItab = *(*uintptr)(unsafe.Pointer(&rIface))

	for i := 0; i < 10; i++ {
		hBuf.Write(&HTTPConfig{})
		gBuf.Write(&GRPCConfig{})
		rBuf.Write(&RepeatConfig{})
	}
}

type BaseConfig struct {
	ID        int
	Start     int
	Name      string `gurlf:"config_name"`
	Type      string `gurlf:"Type"`
	Resp      string `gurlf:"Response"`
	Deps      [6]Dependency
	ExtraDeps []Dependency
	DepsLen   uint8
}

func defBase() *BaseConfig {
	return &BaseConfig{
		Type: "http",
		Name: "http_config",
		Resp: "",
	}
}

func (c *BaseConfig) UnwrapExec() Config             { return c }
func (c *BaseConfig) SetOrig(Config)                 {}
func (c *BaseConfig) Apply(int, int, string, []byte) {}
func (c *BaseConfig) Release()                       {}
func (c *BaseConfig) Clone() Config                  { cp := *c; return &cp }
func (c *BaseConfig) GetName() string                { return c.Name }
func (c *BaseConfig) SetName(nName string)           { c.Name = nName }
func (c *BaseConfig) GetID() int                     { return c.ID }
func (c *BaseConfig) SetID(nID int)                  { c.ID = nID }
func (c *BaseConfig) GetStart() int                  { return c.Start }
func (c *BaseConfig) SetStart(nSt int)               { c.Start = nSt }
func (c *BaseConfig) GetType() string                { return c.Type }
func (c *BaseConfig) SetType(nType string)           { c.Type = nType }
func (c *BaseConfig) GetResp() string                { return c.Resp }
func (c *BaseConfig) SetResp(nResp string)           { c.Resp = nResp }
func (c *BaseConfig) RangeDeps(fn func(d Dependency)) {
	limit := c.DepsLen
	limit = min(limit, 6)

	for i := range limit {
		fn(c.Deps[i])
	}
	for _, d := range c.ExtraDeps {
		fn(d)
	}
}
func (c *BaseConfig) SetDependency(nDep Dependency) {
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

func GetHTTP() (*HTTPConfig, uintptr)    { return hBuf.Read(), hItab }
func (c *HTTPConfig) UnwrapExec() Config { return c }
func (c *HTTPConfig) Release() {
	*c = HTTPConfig{}
	hBuf.Write(c)
}
func (c *HTTPConfig) Clone() Config {
	newCfg := hBuf.Read()
	*newCfg = *c
	newCfg.Url = cloneBytes(c.Url)
	newCfg.Method = cloneBytes(c.Method)
	newCfg.Body = cloneBytes(c.Body)
	newCfg.Headers = cloneBytes(c.Headers)
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

func GetGRPC() (*GRPCConfig, uintptr)    { return gBuf.Read(), gItab }
func (c *GRPCConfig) UnwrapExec() Config { return c }
func (c *GRPCConfig) Release()           { *c = GRPCConfig{}; gBuf.Write(c) }
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
	TargetID int `gurlf:"Target_ID"`
	Orig     Config
}

func GetRepeat() (*RepeatConfig, uintptr) { return rBuf.Read(), rItab }
func (c *RepeatConfig) UnwrapExec() Config {
	if c.Orig == nil {
		return nil
	}
	return c.Orig
}
func (c *RepeatConfig) Release()      { *c = RepeatConfig{}; rBuf.Write(c) }
func (c *RepeatConfig) SetID(nID int) { c.TargetID = nID }
func (c *RepeatConfig) SetOrig(nc Config) {
	if c.Orig != nil {
		c.Orig = nc
	}
}
func (c *RepeatConfig) SetResp(nResp string) {
	if c.Orig != nil {
		c.Orig.SetResp(nResp)
	}
}
func (c *RepeatConfig) Clone() Config {
	newCfg := rBuf.Read()
	*newCfg = *c
	if c.Orig != nil {
		newCfg.Orig = c.Orig.Clone()
	} else {
		newCfg.Orig = nil
	}
	return newCfg
}
func splice(orig, val []byte, start, end int) []byte {
	res := make([]byte, 0, len(orig)+len(val))
	res = append(res, orig[:start]...)
	res = append(res, val...)
	res = append(res, orig[end:]...)
	return res
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}

	nb := make([]byte, len(b))
	copy(nb, b)
	return nb
}
