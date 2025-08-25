package config

import "encoding/json"

type Config interface {
	GetID() string
	GetType() string

	GetHeaders() (json.RawMessage, error)
	SetHeaders(json.RawMessage) error

	GetBody() json.RawMessage
	SetBody(json.RawMessage)

	SetResponse(response string)
	GetResponse() string
}

type HTTPConfig struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Url      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     json.RawMessage   `json:"body,omitempty"`
	Response string            `json:"response,omitempty"`
}

func (h *HTTPConfig) Clone() Config {
	copy := *h
	return &copy
}

func (h *HTTPConfig) GetID() string {
	return h.ID
}

func (h *HTTPConfig) GetType() string {
	return h.Type
}

func (h *HTTPConfig) GetHeaders() (json.RawMessage, error) {
	return json.Marshal(h.Headers)
}

func (h *HTTPConfig) SetHeaders(headers json.RawMessage) error {
	return json.Unmarshal(headers, &h.Headers)
}

func (h *HTTPConfig) GetBody() json.RawMessage {
	return h.Body
}

func (h *HTTPConfig) SetBody(body json.RawMessage) {
	h.Body = body
}

func (h *HTTPConfig) SetResponse(response string) {
	h.Response = response
}

func (h *HTTPConfig) GetResponse() string {
	return h.Response
}

type GRPCConfig struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Target      string            `json:"target"`
	Endpoint    string            `json:"endpoint"`
	Data        json.RawMessage   `json:"data,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Response    string            `json:"response,omitempty"`
	ProtoFiles  []string          `json:"protofiles,omitempty"`
	DialOptions []string          `json:"dialoptions,omitempty"`
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

func (g *GRPCConfig) GetResponse() string {
	return g.Response
}

func (g *GRPCConfig) SetHeaders(md json.RawMessage) error {
	return json.Unmarshal(md, &g.Metadata)
}

func (g *GRPCConfig) GetHeaders() (json.RawMessage, error) {
	return json.Marshal(g.Metadata)
}

func (g *GRPCConfig) SetBody(d json.RawMessage) {
	g.Data = d
}

func (g *GRPCConfig) GetBody() json.RawMessage {
	return g.Data
}
