package config

import "encoding/json"

type Config interface {
	GetID() string
	GetType() string
	SetResponse(response string)
}

type HTTPConfig struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Url      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Body     json.RawMessage   `json:"body,omitempty"`
	Response string            `json:"response,omitempty"`
}

func (h *HTTPConfig) GetID() string {
	return h.ID
}

func (h *HTTPConfig) GetType() string {
	return h.Type
}

func (h *HTTPConfig) SetResponse(response string) {
	h.Response = response
}

type GRPCConfig struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Endpoint string            `json:"endpoint"`
	Data     json.RawMessage   `json:"data,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Response string            `json:"response,omitempty"`
}

func (g *GRPCConfig) GetID() string {
	return g.ID
}

func (g *GRPCConfig) GetType() string {
	return g.Type
}

func (g *GRPCConfig) SetResponse(response string) {
	g.Response = response
}
