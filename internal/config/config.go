package config

import "encoding/json"

type HTTPConfig struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Url      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Body     json.RawMessage   `json:"body,omitempty"`
	Response string            `json:"response,omitempty"`
}

type GRPCConfig struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Endpoint string            `json:"endpoint"`
	Data     json.RawMessage   `json:"data,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Response string            `json:"response,omitempty"`
}
