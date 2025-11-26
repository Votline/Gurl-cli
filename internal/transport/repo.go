package transport

import (
	"net/http"

	"go.uber.org/zap"
)

type HTTPClient struct {
	log *zap.Logger
	ic bool
	cookiePath string
	jar http.CookieJar
	cookies map[string][]*http.Cookie
}
func NewClient(ic bool, ckPath string, log *zap.Logger) *HTTPClient {
	return &HTTPClient{
		ic: ic,
		log: log,
		cookiePath: ckPath,
		cookies: make(map[string][]*http.Cookie),
	}
}

type Result struct {
	Raw *http.Response
	RawBody []byte
	JSON map[string]any
}
