package transport

import (
	"net/http"

	"go.uber.org/zap"

	"Gurl-cli/internal/cookies"
)

type HTTPClient struct {
	log *zap.Logger
	ic bool
	CkCl *cookies.CookiesClient
}
func NewHTTP(ic bool, ckPath string, log *zap.Logger) *HTTPClient {
	return &HTTPClient{
		ic: ic,
		log: log,
		CkCl: cookies.NewClient(ckPath, log),
	}
}
type Result struct {
	Raw *http.Response
	RawBody []byte
	JSON map[string]any
}

type GRPCClient struct {
	log *zap.Logger
}
func NewGRPC(log *zap.Logger) *GRPCClient {
	return &GRPCClient{log: log}
}
