package config

import "encoding/json"

func SetupHTTP() *HTTPConfig {
	return &HTTPConfig{
		ID:     "1",
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
		ID:       "1",
		Type:     "grpc",
		Endpoint: "service.Method",
		Data:     json.RawMessage{},
		Metadata: map[string]string{
			"authorization": "bearer -",
		},
		ProtoFiles: []string{},
		Response: "",
	}
}

func SetupMixed() []any {
	grpc := SetupGRPC()
	grpc.ID = "2"
	return []any{
		SetupHTTP(),
		grpc,
	}
}
