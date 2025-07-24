package config

type HTTPConfig struct {
	Type    string            `json:"type"`
	Url     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body,omitempty"`
}

type GRPCConfig struct {
	Type     string            `json:"type"`
	Endpoint string            `json:"endpoint"`
	Data     interface{}       `json:"data,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
