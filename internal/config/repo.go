package config

type Config interface {
	Clone() Config

	GetID() int
	SetID(int)

	GetType() string
	SetType(string)
}

type BaseConfig struct {
	Name string `gurlf:"config_name"`
	ID   int    `gurlf:"id"`
	Type string `gurlf:"type"`
	Resp string `gurlf:"response"`
}

func defBase() *BaseConfig {
	return &BaseConfig{
		Name: "http_config",
		ID:   0,
		Type: "http",
		Resp: "",
	}
}

func (c *BaseConfig) Clone() Config        { cp := *c; return &cp }
func (c *BaseConfig) GetID() int           { return c.ID }
func (c *BaseConfig) SetID(nID int)        { c.ID = nID }
func (c *BaseConfig) GetType() string      { return c.Type }
func (c *BaseConfig) SetType(nType string) { c.Type = nType }

type HTTPConfig struct {
	BaseConfig
}
func (c *HTTPConfig) Clone() Config        { cp := *c; return &cp }

type GRPCConfig struct {
	BaseConfig
}
func (c *GRPCConfig) Clone() Config        { cp := *c; return &cp }

type RepeatConfig struct {
	BaseConfig
	TargetID int `gurlf:"target_id"`
}
