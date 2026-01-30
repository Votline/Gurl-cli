package config

import (
	"unsafe"

	"gcli/internal/buffer"
)

const (
	NoRepeatConfig     int    = -1
	DataFromFile       int    = -2
	MaxLen             int    = -3
	FlagUseFileCookies uint32 = 1 << iota
)

type Dependency struct {
	TargetID int
	Start    int
	End      int
	Key      string
	InsTp    string
}

type Config interface {
	Clone() Config
	Release()
	ReleaseClone()

	GetName() string
	SetID(int)
	GetType() string

	GetEnd() int
	SetEnd(int)

	Update([]byte, []byte)
	GetRaw(string) []byte

	UnwrapExec() Config

	RangeDeps(func(d Dependency))
	SetDependency(Dependency)

	Apply(int, int, string, []byte)

	HasFlag(uint32) bool
	SetFlag(uint32)
}

var (
	hBuf    = buffer.NewRb[*HTTPConfig]()
	gBuf    = buffer.NewRb[*GRPCConfig]()
	rBuf    = buffer.NewRb[*RepeatConfig]()
	hItab   uintptr
	gItab   uintptr
	rItab   uintptr
	hClBuf  = buffer.NewRb[*HTTPConfig]()
	gClBuf  = buffer.NewRb[*GRPCConfig]()
	rClBuf  = buffer.NewRb[*RepeatConfig]()
	hItabCl uintptr
	gItabCl uintptr
	rItabCl uintptr
)

func Init() {
	hBuf = buffer.NewRb[*HTTPConfig]()
	gBuf = buffer.NewRb[*GRPCConfig]()
	rBuf = buffer.NewRb[*RepeatConfig]()
	hClBuf = buffer.NewRb[*HTTPConfig]()
	gClBuf = buffer.NewRb[*GRPCConfig]()
	rClBuf = buffer.NewRb[*RepeatConfig]()

	var hIface Config = &HTTPConfig{}
	hItab = *(*uintptr)(unsafe.Pointer(&hIface))

	var gIface Config = &GRPCConfig{}
	gItab = *(*uintptr)(unsafe.Pointer(&gIface))

	var rIface Config = &RepeatConfig{}
	rItab = *(*uintptr)(unsafe.Pointer(&rIface))

	var hIfaceCl Config = &HTTPConfig{}
	hItabCl = *(*uintptr)(unsafe.Pointer(&hIfaceCl))

	var gIfaceCl Config = &GRPCConfig{}
	gItabCl = *(*uintptr)(unsafe.Pointer(&gIfaceCl))

	var rIfaceCl Config = &RepeatConfig{}
	rItabCl = *(*uintptr)(unsafe.Pointer(&rIfaceCl))

	for i := 0; i < 10; i++ {
		hBuf.Write(&HTTPConfig{})
		gBuf.Write(&GRPCConfig{})
		rBuf.Write(&RepeatConfig{})
		hClBuf.Write(&HTTPConfig{})
		gClBuf.Write(&GRPCConfig{})
		rClBuf.Write(&RepeatConfig{})
	}
}

func Alloc(cfg Config) Config {
	switch v := cfg.(type) {
	case *HTTPConfig:
		cp := new(HTTPConfig)
		*cp = *v
		cp.URL = cloneBytes(v.URL)
		cp.Method = cloneBytes(v.Method)
		cp.Body = cloneBytes(v.Body)
		cp.Headers = cloneBytes(v.Headers)
		return cp
	case *GRPCConfig:
		cp := new(GRPCConfig)
		*cp = *v
		return cp
	case *RepeatConfig:
		cp := new(RepeatConfig)
		*cp = *v
		return cp
	}
	return nil
}

type BaseConfig struct {
	ID        int
	End       int
	Len       int
	Name      string `gurlf:"config_name"`
	Type      string `gurlf:"Type"`
	Resp      []byte `gurlf:"Response"`
	Deps      [6]Dependency
	ExtraDeps []Dependency
	DepsLen   uint8
	flag      uint32
}

func defBase() *BaseConfig {
	return &BaseConfig{
		Type: "http",
		Name: "http_config",
		Resp: nil,
	}
}
func (c *BaseConfig) Release()                       {}
func (c *BaseConfig) ReleaseClone()                  {}
func (c *BaseConfig) Update(raw, cookie []byte)      {}
func (c *BaseConfig) Apply(int, int, string, []byte) {}
func (c *BaseConfig) GetRaw(key string) []byte       { return nil }
func (c *BaseConfig) Clone() Config                  { cp := *c; return &cp }
func (c *BaseConfig) GetName() string                { return c.Name }
func (c *BaseConfig) SetID(nID int)                  { c.ID = nID }
func (c *BaseConfig) GetType() string                { return c.Type }
func (c *BaseConfig) GetEnd() int                    { return c.End }
func (c *BaseConfig) SetEnd(nEnd int)                { c.End = nEnd }
func (c *BaseConfig) UnwrapExec() Config             { return c }
func (c *BaseConfig) HasFlag(f uint32) bool          { return c.flag&f != 0 }
func (c *BaseConfig) SetFlag(f uint32)               { c.flag |= f }

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
	URL     []byte `gurlf:"URL"`
	Method  []byte `gurlf:"Method"`
	Body    []byte `gurlf:"Body"`
	Headers []byte `gurlf:"Headers"`
	BaseConfig
	CookieIn  []byte `gurlf:"CookieIn"`
	CookieOut []byte `gurlf:"CookieOut"`
}

func GetHTTP() (*HTTPConfig, uintptr)    { return hBuf.Read(), hItab }
func (c *HTTPConfig) GetCookie() []byte  { return c.CookieIn }
func (c *HTTPConfig) SetCookie(b []byte) { c.CookieOut = b }
func (c *HTTPConfig) UnwrapExec() Config { return c }
func (c *HTTPConfig) Release()           { *c = HTTPConfig{}; hBuf.Write(c) }
func (c *HTTPConfig) ReleaseClone()      { *c = HTTPConfig{}; hClBuf.Write(c) }
func (c *HTTPConfig) Clone() Config {
	newCfg := hClBuf.Read()
	*newCfg = *c
	newCfg.URL = cloneBytes(c.URL)
	newCfg.Method = cloneBytes(c.Method)
	newCfg.Body = cloneBytes(c.Body)
	newCfg.Headers = cloneBytes(c.Headers)
	return newCfg
}

func (c *HTTPConfig) Update(res, cks []byte) {
	tmp := make([]byte, len(res))
	copy(tmp, res)
	c.Resp = tmp
	c.CookieOut = cks
}

func (c *HTTPConfig) GetRaw(key string) []byte {
	switch key {
	case "URL":
		return c.URL
	case "Method":
		return c.Method
	case "Body":
		return c.Body
	case "Headers":
		return c.Headers
	case "Cookie":
		return c.CookieIn
	}
	return nil
}

func (c *HTTPConfig) Apply(start, end int, key string, val []byte) {
	switch key {
	case "URL":
		c.URL = splice(c.URL, val, start, end)
	case "Method":
		c.Method = splice(c.Method, val, start, end)
	case "Body":
		c.Body = splice(c.Body, val, start, end)
	case "Headers":
		c.Headers = splice(c.Headers, val, start, end)
	case "Cookie":
		c.CookieIn = splice(c.CookieIn, val, start, end)
	}
}

type GRPCConfig struct {
	Target      []byte `gurlf:"Target"`
	Endpoint    []byte `gurlf:"Endpoint"`
	Data        []byte `gurlf:"Data"`
	Metadata    []byte `gurlf:"Metadata"`
	ProtoPath   []byte `gurlf:"ProtoPath"`
	ImportPaths []byte `gurlf:"ImportPaths"`
	DialOpts    []byte `gurlf:"DialOpts"`
	BaseConfig
}

func GetGRPC() (*GRPCConfig, uintptr)    { return gBuf.Read(), gItab }
func (c *GRPCConfig) UnwrapExec() Config { return c }
func (c *GRPCConfig) Release()           { *c = GRPCConfig{}; gBuf.Write(c) }
func (c *GRPCConfig) ReleaseClone()      { *c = GRPCConfig{}; gClBuf.Write(c) }
func (c *GRPCConfig) Clone() Config {
	newCfg := gClBuf.Read()
	*newCfg = *c
	return newCfg
}

func (c *GRPCConfig) Update(res, cks []byte) {
	tmp := make([]byte, len(res))
	copy(tmp, res)
	c.Resp = tmp
}

func (c *GRPCConfig) GetRaw(key string) []byte {
	switch key {
	case "Target":
		return c.Target
	case "Endpoint":
		return c.Endpoint
	case "Data":
		return c.Data
	case "Metadata":
		return c.Metadata
	case "ProtoPath":
		return c.ProtoPath
	case "ImportPaths":
		return c.ImportPaths
	case "DialOpts":
		return c.DialOpts
	}
	return nil
}

func (c *GRPCConfig) Apply(start, end int, key string, val []byte) {
	switch key {
	case "Target":
		c.Target = splice(c.Target, val, start, end)
	case "Endpoint":
		c.Endpoint = splice(c.Endpoint, val, start, end)
	case "Data":
		c.Data = splice(c.Data, val, start, end)
	case "Metadata":
		c.Metadata = splice(c.Metadata, val, start, end)
	case "ProtoPath":
		c.ProtoPath = splice(c.ProtoPath, val, start, end)
	case "ImportPaths":
		c.ImportPaths = splice(c.ImportPaths, val, start, end)
	case "DialOpts":
		c.DialOpts = splice(c.DialOpts, val, start, end)
	}
}

type RepeatConfig struct {
	TargetID int    `gurlf:"Target_ID"`
	Replace  []byte `gurlf:"Replace"`
	Orig     Config
	BaseConfig
}

func GetRepeat() (*RepeatConfig, uintptr) { return rBuf.Read(), rItab }
func (c *RepeatConfig) UnwrapExec() Config {
	if c.Orig == nil {
		return nil
	}
	return c.Orig
}
func (c *RepeatConfig) SetID(nID int) { c.TargetID = nID }
func (c *RepeatConfig) Release() {
	*c = RepeatConfig{}
	rBuf.Write(c)
}

func (c *RepeatConfig) ReleaseClone() {
	if c.Orig != nil {
		c.Orig.ReleaseClone()
	}
	*c = RepeatConfig{}
	rClBuf.Write(c)
}

func (c *RepeatConfig) Clone() Config {
	newCfg := rClBuf.Read()
	*newCfg = *c

	if c.Orig != nil {
		newCfg.Orig = c.Orig.Clone()
	}

	return newCfg
}

func (c *RepeatConfig) Update(res, cks []byte) {
	tmp := make([]byte, len(res))
	copy(tmp, res)
	c.Resp = tmp
	c.Orig.Update(nil, cks)
}

func (c *RepeatConfig) GetRaw(key string) []byte {
	switch key {
	case "Replace":
		return c.Replace
	}
	return nil
}

func (c *RepeatConfig) Apply(start, end int, key string, val []byte) {
	switch key {
	case "Replace":
		c.Replace = splice(c.Replace, val, start, end)
	}
}

func splice(orig, val []byte, start, end int) []byte {
	if end == MaxLen {
		end = len(orig)
	}
	if start < 0 || end < 0 || start >= len(orig) || end > len(orig) || start == end {
		return nil
	}
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
