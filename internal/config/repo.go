package config

type Config interface {
	GetID() int
	SetID(int)

	GetType() string
	SetType(string)
}

type BaseConfig struct {
	Name string `gurlf:"config_name"`
	ID   int    `gurlf:"id"`
	Type string `gurlf:"type"`
}
func defBase() *BaseConfig {
	return &BaseConfig{
		Name: "http_config",
		ID: 0,
		Type: "http",
	}
}

func (c *BaseConfig) GetID() int           { return c.ID }
func (c *BaseConfig) SetId(nID int)        { c.ID = nID }
func (c *BaseConfig) GetType() string      { return c.Type }
func (c *BaseConfig) SetType(nType string) { c.Type = nType }

type HTTPConfig struct {
	BaseConfig
}

type GRPCConfig struct {
	BaseConfig
}

type RepeatConfig struct {
	BaseConfig
}
