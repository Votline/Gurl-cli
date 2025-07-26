package config

func SetupHTTP() *HTTPConfig {
	return &HTTPConfig{
		Type:   "http",
		Url:    "-",
		Method: "-",
		Headers: map[string]string{
			"Authorization": "Bearer -",
			"Content-Type":  "application/json",
		},
		Body: map[string]any{},
	}
}

func SetupGRPC() *GRPCConfig {
	return &GRPCConfig{
		Type:     "grpc",
		Endpoint: "service.Method",
		Data:     map[string]any{},
		Metadata: map[string]string{
			"authorization": "bearer -",
		},
	}
}

func SetupMixed() []any {
	return []any{
		SetupHTTP(),
		SetupGRPC(),
	}
}
