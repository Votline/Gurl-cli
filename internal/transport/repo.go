// Package transport repo.go contains structs for transport package.
package transport

import (
	"crypto/tls"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// builderPool is a sync.Pool for strings.Builder
var builderPool = sync.Pool{
	New: func() any {
		return new(strings.Builder)
	},
}

// Status is a struct for response status.
type Status struct {
	// Code is a response code.
	Code int

	// Message is a response message.
	Message string

	// ConfigType is a type of config.
	ConfigType string
}

// Result is a struct for response.
type Result struct {
	// Info is a response status.
	Info Status

	// CfgID is a id of config.
	CfgID int

	// IsJSON is a flag for JSON response.
	IsJSON bool

	// Raw is a raw response.
	Raw []byte

	// Cookie is a raw cookie.
	Cookie []byte
}

// Transport is a struct for transport package.
type Transport struct {
	// jar is a map for cookies.
	jar map[string]string

	// cl is a http.Client.
	cl *http.Client

	// log is a zap.Logger.
	log *zap.Logger
}

// NewTransport accepts callback for result and zap.Logger.
// It returns new Transport.
func NewTransport(putRes func(*Result), log *zap.Logger) *Transport {
	for i := 0; i < 10; i++ {
		putRes(&Result{})
	}
	client := &http.Client{
		Jar: nil,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	return &Transport{jar: map[string]string{}, cl: client, log: log}
}
