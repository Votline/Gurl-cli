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

func SetupHTTP() *HTTPConfig {
	return &HTTPConfig{
		Type: "http",
		Url: "-",
		Method: "-",
		Headers: map[string]string{
			"Authorization": "Bearer -",
			"Content-Type": "application/json",
		},
	}
}

func SetupGRPC() *GRPCConfig {
	return &GRPCConfig{
		Type: "grpc",
		Endpoint: "service.Method",
		Data: map[string]any{},
		Metadata: map[string]string{
			"authorization": "bearer -",
		},
	}
}
