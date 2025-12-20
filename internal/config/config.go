package config

import (
	"encoding/json"

	"go.uber.org/zap"
)

type Parser struct{
	log *zap.Logger
}
func NewParser(log *zap.Logger) *Parser {
	return &Parser{log: log}
}

type Config interface {
	GetID() string
	SetID(string)

	GetType() string

	GetUrl() (string)
	SetUrl(string)

	GetHeaders() (json.RawMessage, error)
	SetHeaders(json.RawMessage) error

	GetBody() json.RawMessage
	SetBody(json.RawMessage)

	GetResponse() string
	SetResponse(response string)

	GetReplace() map[string]any

	GetCookies() json.RawMessage
	SetCookies(json.RawMessage)
}

type HTTPConfig struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Url      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     json.RawMessage   `json:"body,omitempty"`
	Response string            `json:"response,omitempty"`
	Cookies  json.RawMessage `json:"cookies"`
}

func (h *HTTPConfig) Clone() Config {
	copy := *h
	return &copy
}

func (h *HTTPConfig) GetID() string {
	return h.ID
}
func (h *HTTPConfig) SetID(newID string) {
	h.ID = newID
}

func (h *HTTPConfig) GetType() string {
	return h.Type
}

func (h *HTTPConfig) GetUrl() string {
	return h.Url
}
func (h *HTTPConfig) SetUrl(url string) {
	h.Url = url
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

func (h *HTTPConfig) GetResponse() string {
	return h.Response
}
func (h *HTTPConfig) SetResponse(response string) {
	h.Response = response
}

func (h *HTTPConfig) GetReplace() map[string]any {
	return nil
}

func (h *HTTPConfig) GetCookies() json.RawMessage {
	return h.Cookies
}
func (h *HTTPConfig) SetCookies(cks json.RawMessage) {
	h.Cookies = cks
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
func (g *GRPCConfig) SetID(newID string) {
	g.ID = newID
}

func (g *GRPCConfig) GetType() string {
	return g.Type
}

func (g *GRPCConfig) GetUrl() string {
	return ""
}
func (g *GRPCConfig) SetUrl(url string) {
}

func (g *GRPCConfig) GetResponse() string {
	return g.Response
}
func (g *GRPCConfig) SetResponse(response string) {
	g.Response = response
}

func (g *GRPCConfig) GetHeaders() (json.RawMessage, error) {
	return json.Marshal(g.Metadata)
}
func (g *GRPCConfig) SetHeaders(md json.RawMessage) error {
	return json.Unmarshal(md, &g.Metadata)
}

func (g *GRPCConfig) GetBody() json.RawMessage {
	return g.Data
}
func (g *GRPCConfig) SetBody(d json.RawMessage) {
	g.Data = d
}

func (g *GRPCConfig) GetReplace() map[string]any {
	return nil
}

func (h *GRPCConfig) GetCookies() json.RawMessage {
	return nil
}
func (h *GRPCConfig) SetCookies(cks json.RawMessage) {
}

type RepeatedConfig struct {
	Type      string         `json:"type"`
	RepID     string         `json:"repeated_id"`
	Replace   map[string]any `json:"replace,omitempty"`
}

func (r *RepeatedConfig) GetID() string {
	return r.RepID
}
func (r *RepeatedConfig) SetID(string){
}

func (r *RepeatedConfig) GetType() string {
	return r.Type
}

func (r *RepeatedConfig) GetUrl() string {
	return ""
}

func (r *RepeatedConfig) SetUrl(url string) {
}

func (r *RepeatedConfig) GetHeaders() (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (r *RepeatedConfig) SetHeaders(headers json.RawMessage) error {
	return nil
}

func (r *RepeatedConfig) GetBody() json.RawMessage {
	return json.RawMessage(`{}`)
}

func (r *RepeatedConfig) SetBody(body json.RawMessage) {
}

func (r *RepeatedConfig) GetResponse() string {
	return ""
}

func (r *RepeatedConfig) SetResponse(response string) {
}

func (r *RepeatedConfig) GetReplace() map[string]any {
	return r.Replace
}

func (h *RepeatedConfig) GetCookies() json.RawMessage {
	return nil
}
func (h *RepeatedConfig) SetCookies(cks json.RawMessage) {
}

