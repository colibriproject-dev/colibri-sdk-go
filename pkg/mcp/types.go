package mcp

import (
	"context"
	"encoding/json"
)

// Message representa uma mensagem JSON-RPC 2.0 que pode ser um Request, Response ou Notification.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Request representa uma requisição JSON-RPC 2.0.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response representa uma resposta JSON-RPC 2.0.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Notification representa uma notificação JSON-RPC 2.0.
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCError representa um objeto de erro seguindo a especificação JSON-RPC 2.0.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return e.Message
}

// ServerInfo contém metadados sobre a implementação do servidor MCP.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities define os recursos opcionais suportados pelo servidor MCP.
type ServerCapabilities struct {
	Tools     *ToolCapabilities     `json:"tools,omitempty"`
	Resources *ResourceCapabilities `json:"resources,omitempty"`
	Logging   *LoggingCapabilities  `json:"logging,omitempty"`
}

// ToolCapabilities define os recursos relacionados a ferramentas.
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapabilities define os recursos relacionados a recursos (resources).
type ResourceCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapabilities é um placeholder para capacidades de log futuras.
type LoggingCapabilities struct {
}

// Tool descreve uma ferramenta que o servidor disponibiliza para o cliente.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema representa o JSON Schema que define os parâmetros de entrada de uma ferramenta.
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// LogLevel define a severidade de um log.
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelNotice  LogLevel = "notice"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

// ToolHandler é a função que processa a execução de uma ferramenta.
type ToolHandler func(ctx context.Context, params any) (any, error)
