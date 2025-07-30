package config

import "encoding/json"

func SetupHTTP() *HTTPConfig {
	return &HTTPConfig{
		Type:   "http",
		Url:    "-",
		Method: "-",
		Headers: map[string]string{
			"Authorization": "Bearer -",
			"Content-Type":  "application/json",
		},
		Body: json.RawMessage{},
		Response: "",
	}
}

func SetupGRPC() *GRPCConfig {
	return &GRPCConfig{
		Type:     "grpc",
		Endpoint: "service.Method",
		Data:     json.RawMessage{},
		Metadata: map[string]string{
			"authorization": "bearer -",
		},
		Response: "",
	}
}

func SetupMixed() []any {
	return []any{
		SetupHTTP(),
		SetupGRPC(),
	}
}
