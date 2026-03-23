package config

import (
	"unsafe"

	"gcli/internal/buffer"
)

const (
	NoRepeatConfig     int    = -1
	DataFromFile       int    = -2
	MaxLen             int    = -3
	RandomData         int    = -4
	DataFromVariable   int    = -5
	FlagUseFileCookies uint32 = 1
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

	GetID() int
	SetID(int)

	GetType() string

	GetWait() []byte
	SetWait([]byte)

	GetExpect() []byte
	SetExpect([]byte)

	GetEnd() int
	SetEnd(int)

	Update([]byte, []byte)
	GetRaw(string) []byte

	UnwrapExec() Config

	RangeDeps(func(d Dependency))
	GetDepsLen() uint8
	SetDependency(Dependency)

	Apply(int, int, string, []byte)

	HasFlag(uint32) bool
	SetFlag(uint32)
}

var (
	hBuf    = buffer.NewRb[*HTTPConfig]()
	gBuf    = buffer.NewRb[*GRPCConfig]()
	rBuf    = buffer.NewRb[*RepeatConfig]()
	iBuf    = buffer.NewRb[*ImportConfig]()
	hItab   uintptr
	gItab   uintptr
	rItab   uintptr
	iItab   uintptr
	hClBuf  = buffer.NewRb[*HTTPConfig]()
	gClBuf  = buffer.NewRb[*GRPCConfig]()
	rClBuf  = buffer.NewRb[*RepeatConfig]()
	iClBuf  = buffer.NewRb[*ImportConfig]()
	hItabCl uintptr
	gItabCl uintptr
	rItabCl uintptr
	iItabCl uintptr
)

func Init() {
	hBuf = buffer.NewRb[*HTTPConfig]()
	gBuf = buffer.NewRb[*GRPCConfig]()
	rBuf = buffer.NewRb[*RepeatConfig]()
	iBuf = buffer.NewRb[*ImportConfig]()
	hClBuf = buffer.NewRb[*HTTPConfig]()
	gClBuf = buffer.NewRb[*GRPCConfig]()
	rClBuf = buffer.NewRb[*RepeatConfig]()
	iClBuf = buffer.NewRb[*ImportConfig]()

	var hIface Config = &HTTPConfig{}
	hItab = *(*uintptr)(unsafe.Pointer(&hIface))

	var gIface Config = &GRPCConfig{}
	gItab = *(*uintptr)(unsafe.Pointer(&gIface))

	var rIface Config = &RepeatConfig{}
	rItab = *(*uintptr)(unsafe.Pointer(&rIface))

	var iIface Config = &ImportConfig{}
	iItab = *(*uintptr)(unsafe.Pointer(&iIface))

	var hIfaceCl Config = &HTTPConfig{}
	hItabCl = *(*uintptr)(unsafe.Pointer(&hIfaceCl))

	var gIfaceCl Config = &GRPCConfig{}
	gItabCl = *(*uintptr)(unsafe.Pointer(&gIfaceCl))

	var rIfaceCl Config = &RepeatConfig{}
	rItabCl = *(*uintptr)(unsafe.Pointer(&rIfaceCl))

	var iIfaceCl Config = &ImportConfig{}
	iItabCl = *(*uintptr)(unsafe.Pointer(&iIfaceCl))

	for i := 0; i < 10; i++ {
		hBuf.Write(&HTTPConfig{})
		gBuf.Write(&GRPCConfig{})
		rBuf.Write(&RepeatConfig{})
		iBuf.Write(&ImportConfig{})
		hClBuf.Write(&HTTPConfig{})
		gClBuf.Write(&GRPCConfig{})
		rClBuf.Write(&RepeatConfig{})
		iClBuf.Write(&ImportConfig{})
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
		cp.CookieIn = cloneBytes(v.CookieIn)
		cp.CookieOut = cloneBytes(v.CookieOut)
		cp.Wait = cloneBytes(v.Wait)
		cp.Expect = cloneBytes(v.Expect)
		return cp
	case *GRPCConfig:
		cp := new(GRPCConfig)
		*cp = *v
		cp.Target = cloneBytes(v.Target)
		cp.Endpoint = cloneBytes(v.Endpoint)
		cp.Data = cloneBytes(v.Data)
		cp.ProtoPath = cloneBytes(v.ProtoPath)
		cp.ImportPaths = cloneBytes(v.ImportPaths)
		cp.DialOpts = cloneBytes(v.DialOpts)
		cp.Wait = cloneBytes(v.Wait)
		cp.Expect = cloneBytes(v.Expect)
		return cp
	case *RepeatConfig:
		cp := new(RepeatConfig)
		*cp = *v
		cp.Wait = cloneBytes(v.Wait)
		cp.Expect = cloneBytes(v.Expect)
		return cp
	case *ImportConfig:
		cp := new(ImportConfig)
		*cp = *v
		cp.Wait = cloneBytes(v.Wait)
		cp.Expect = cloneBytes(v.Expect)
		cp.TargetPath = v.TargetPath
		cp.Vars = cloneBytes(v.Vars)
		return cp
	}
	return nil
}

type BaseConfig struct {
	ID        int `gurlf:"ID"`
	End       int
	Len       int
	Name      string `gurlf:"config_name"`
	Type      string `gurlf:"Type"`
	Wait      []byte `gurlf:"Wait,omitempty"`
	Expect    []byte `gurlf:"Expect,omitempty"`
	Resp      []byte `gurlf:"Response,omitempty"`
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
func (c *BaseConfig) GetDepsLen() uint8              { return c.DepsLen }
func (c *BaseConfig) GetRaw(key string) []byte       { return nil }
func (c *BaseConfig) GetName() string                { return c.Name }
func (c *BaseConfig) GetWait() []byte                { return c.Wait }
func (c *BaseConfig) SetWait(nWait []byte)           { c.Wait = nWait }
func (c *BaseConfig) GetExpect() []byte              { return c.Expect }
func (c *BaseConfig) SetExpect(nExpect []byte)       { c.Expect = nExpect }
func (c *BaseConfig) SetID(nID int)                  { c.ID = nID }
func (c *BaseConfig) GetID() int                     { return c.ID }
func (c *BaseConfig) GetType() string                { return c.Type }
func (c *BaseConfig) GetEnd() int                    { return c.End }
func (c *BaseConfig) SetEnd(nEnd int)                { c.End = nEnd }
func (c *BaseConfig) UnwrapExec() Config             { return c }
func (c *BaseConfig) HasFlag(f uint32) bool          { return c.flag&f != 0 }
func (c *BaseConfig) SetFlag(f uint32)               { c.flag |= f }
func (c *BaseConfig) Clone() Config {
	cp := *c
	cp.Wait = cloneBytes(c.Wait)
	cp.Expect = cloneBytes(c.Expect)
	return &cp
}

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
	limit := min(c.DepsLen, 6)

	for i := range limit {
		if c.Deps[i].Start == nDep.Start && c.Deps[i].End == nDep.End && c.Deps[i].Key == nDep.Key {
			return
		}
	}
	for _, d := range c.ExtraDeps {
		if d.Start == nDep.Start && d.End == nDep.End && d.Key == nDep.Key {
			return
		}
	}

	if c.DepsLen < 6 {
		c.Deps[c.DepsLen] = nDep
	} else {
		c.ExtraDeps = append(c.ExtraDeps, nDep)
	}
	c.DepsLen++
}

type HTTPConfig struct {
	URL     []byte `gurlf:"URL"`
	Method  []byte `gurlf:"Method,omitempty"`
	Body    []byte `gurlf:"Body,omitempty"`
	Headers []byte `gurlf:"Headers,omitempty"`
	BaseConfig
	CookieIn  []byte `gurlf:"CookieIn,omitempty"`
	CookieOut []byte `gurlf:"CookieOut,omitempty"`
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
	newCfg.CookieIn = cloneBytes(c.CookieIn)
	newCfg.CookieOut = cloneBytes(c.CookieOut)
	newCfg.Wait = cloneBytes(c.Wait)
	newCfg.Expect = cloneBytes(c.Expect)
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
	case "Wait":
		return c.Wait
	case "Expect":
		return c.Expect
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
	case "Wait":
		c.Wait = splice(c.Wait, val, start, end)
	case "Expect":
		c.Expect = splice(c.Expect, val, start, end)
	}
}

type GRPCConfig struct {
	Target      []byte `gurlf:"Target"`
	Endpoint    []byte `gurlf:"Endpoint"`
	Data        []byte `gurlf:"Data,omitempty"`
	Metadata    []byte `gurlf:"Metadata,omitempty"`
	ProtoPath   []byte `gurlf:"ProtoPath,omitempty"`
	ImportPaths []byte `gurlf:"ImportPaths,omitempty"`
	DialOpts    []byte `gurlf:"DialOpts,omitempty"`
	BaseConfig
}

func GetGRPC() (*GRPCConfig, uintptr)    { return gBuf.Read(), gItab }
func (c *GRPCConfig) UnwrapExec() Config { return c }
func (c *GRPCConfig) Release()           { *c = GRPCConfig{}; gBuf.Write(c) }
func (c *GRPCConfig) ReleaseClone()      { *c = GRPCConfig{}; gClBuf.Write(c) }
func (c *GRPCConfig) Clone() Config {
	newCfg := gClBuf.Read()
	*newCfg = *c
	newCfg.Target = cloneBytes(c.Target)
	newCfg.Endpoint = cloneBytes(c.Endpoint)
	newCfg.Data = cloneBytes(c.Data)
	newCfg.ProtoPath = cloneBytes(c.ProtoPath)
	newCfg.ImportPaths = cloneBytes(c.ImportPaths)
	newCfg.DialOpts = cloneBytes(c.DialOpts)
	newCfg.Wait = cloneBytes(c.Wait)
	newCfg.Expect = cloneBytes(c.Expect)
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
	case "Wait":
		return c.Wait
	case "Expect":
		return c.Expect
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
	case "Wait":
		c.Wait = splice(c.Wait, val, start, end)
	case "Expect":
		c.Expect = splice(c.Expect, val, start, end)
	}
}

type RepeatConfig struct {
	TargetID int    `gurlf:"TargetID"`
	Replace  []byte `gurlf:"Replace,omitempty"`
	Orig     Config
	BaseConfig
}

func GetRepeat() (*RepeatConfig, uintptr)   { return rBuf.Read(), rItab }
func (c *RepeatConfig) SetTargetID(nID int) { c.TargetID = nID }

func (c *RepeatConfig) UnwrapExec() Config {
	if c.Orig == nil {
		return nil
	}
	return c.Orig
}

func (c *RepeatConfig) RangeDeps(fn func(d Dependency)) {
	if c.Orig != nil {
		c.Orig.RangeDeps(fn)
	} else {
		c.BaseConfig.RangeDeps(fn)
	}
}

func (c *RepeatConfig) GetDepsLen() uint8 {
	if c.Orig != nil {
		return c.Orig.GetDepsLen()
	}
	return c.BaseConfig.GetDepsLen()
}
func (c *RepeatConfig) SetID(nID int) { c.ID = nID }
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
	newCfg.Wait = cloneBytes(c.Wait)
	newCfg.Expect = cloneBytes(c.Expect)

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
	case "Wait":
		return c.Wait
	case "Expect":
		return c.Expect
	default:
		if c.Orig != nil {
			return c.Orig.GetRaw(key)
		}
	}
	return nil
}

func (c *RepeatConfig) Apply(start, end int, key string, val []byte) {
	switch key {
	case "Replace":
		c.Replace = splice(c.Replace, val, start, end)
	case "Wait":
		c.Wait = splice(c.Wait, val, start, end)
	case "Expect":
		c.Expect = splice(c.Expect, val, start, end)
	default:
		if c.Orig != nil {
			c.Orig.Apply(start, end, key, val)
		}
	}
}

type ImportConfig struct {
	TargetPath string `gurlf:"TargetPath"`
	Vars       []byte `gurlf:"Variables,omitempty"`
	BaseConfig
}

func GetImport() (*ImportConfig, uintptr)  { return iBuf.Read(), iItab }
func (c *ImportConfig) Release()           { *c = ImportConfig{}; iBuf.Write(c) }
func (c *ImportConfig) UnwrapExec() Config { return c }
func (c *ImportConfig) Clone() Config {
	newCfg := iClBuf.Read()
	*newCfg = *c
	newCfg.Wait = cloneBytes(c.Wait)
	newCfg.Expect = cloneBytes(c.Expect)
	newCfg.TargetPath = c.TargetPath
	newCfg.Vars = cloneBytes(c.Vars)
	return newCfg
}
func (c *ImportConfig) ReleaseClone() { *c = ImportConfig{}; iClBuf.Write(c) }

func (c *ImportConfig) Update(res, cks []byte) {
	tmp := make([]byte, len(res))
	copy(tmp, res)
	c.Resp = tmp
}

func (c *ImportConfig) GetRaw(key string) []byte {
	switch key {
	case "Wait":
		return c.Wait
	case "Expect":
		return c.Expect
	case "Variables":
		return c.Vars
	}
	return nil
}

func (c *ImportConfig) Apply(start, end int, key string, val []byte) {
	switch key {
	case "Wait":
		c.Wait = splice(c.Wait, val, start, end)
	case "Expect":
		c.Expect = splice(c.Expect, val, start, end)
	case "Variables":
		c.Vars = splice(c.Vars, val, start, end)
	}
}

func splice(orig, val []byte, start, end int) []byte {
	if len(orig) == 0 {
		return val
	}

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
