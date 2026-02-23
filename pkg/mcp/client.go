package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// Client gerencia a conexão ativa com um servidor MCP.
type Client struct {
	transport       Transport
	requestID       int64
	pendingRequests map[any]chan Message
	mu              sync.Mutex
	serverInfo      ServerInfo
	capabilities    ServerCapabilities
	cancel          context.CancelFunc
}

// NewClient cria uma nova instância de um cliente MCP.
func NewClient(t Transport) *Client {
	return &Client{
		transport:       t,
		pendingRequests: make(map[any]chan Message),
	}
}

// Connect inicia a conexão e realiza o handshake inicial com o servidor.
func (c *Client) Connect(ctx context.Context) error {
	// Inicia o loop de recepção de mensagens
	receiveCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.listen(receiveCtx)

	// 1. Enviar initialize
	initParams := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"logging": struct{}{},
		},
		"clientInfo": map[string]string{
			"name":    "colibri-sdk-go",
			"version": "0.1.0",
		},
	}

	resp, err := c.Call(ctx, "initialize", initParams)
	if err != nil {
		c.cancel()
		return fmt.Errorf("falha no initialize: %w", err)
	}

	if resp.Error != nil {
		c.cancel()
		return fmt.Errorf("erro do servidor no initialize: %s", resp.Error.Message)
	}

	// Processar resposta do initialize
	var initResult struct {
		ProtocolVersion string             `json:"protocolVersion"`
		Capabilities    ServerCapabilities `json:"capabilities"`
		ServerInfo      ServerInfo         `json:"serverInfo"`
	}

	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		c.cancel()
		return fmt.Errorf("falha ao decodificar resposta do initialize: %w", err)
	}

	c.serverInfo = initResult.ServerInfo
	c.capabilities = initResult.Capabilities

	// 2. Enviar notifications/initialized
	initializedNotification := Message{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	if err := c.transport.Send(initializedNotification); err != nil {
		c.cancel()
		return fmt.Errorf("falha ao enviar notificação initialized: %w", err)
	}

	return nil
}

// Call envia uma requisição JSON-RPC e aguarda a resposta.
func (c *Client) Call(ctx context.Context, method string, params any) (*Message, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar parâmetros: %w", err)
	}

	msg := Message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}

	respChan := make(chan Message, 1)
	c.mu.Lock()
	c.pendingRequests[id] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pendingRequests, id)
		c.mu.Unlock()
	}()

	if err := c.transport.Send(msg); err != nil {
		return nil, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	select {
	case resp := <-respChan:
		return &resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ListTools solicita ao servidor a lista de ferramentas disponíveis.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.Call(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar ferramentas: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("erro do servidor ao listar ferramentas: %s", resp.Error.Message)
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("falha ao decodificar lista de ferramentas: %w", err)
	}

	return result.Tools, nil
}

// CallTool executa uma ferramenta no servidor.
func (c *Client) CallTool(ctx context.Context, name string, args any) (*Message, error) {
	callParams := map[string]any{
		"name":      name,
		"arguments": args,
	}

	resp, err := c.Call(ctx, "tools/call", callParams)
	if err != nil {
		return nil, fmt.Errorf("falha ao chamar ferramenta %s: %w", name, err)
	}

	return resp, nil
}

// Close encerra a conexão do cliente.
func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.transport.Close()
}

func (c *Client) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := c.transport.Receive()
			if err != nil {
				if err == io.EOF {
					return
				}
				// Em caso de erro de leitura, encerramos o loop
				return
			}

			c.handleIncomingMessage(msg)
		}
	}
}

func (c *Client) handleIncomingMessage(msg Message) {
	if msg.ID != nil {
		c.mu.Lock()
		ch, ok := c.pendingRequests[msg.ID]
		c.mu.Unlock()

		if ok {
			ch <- msg
		}
	} else {
		// Tratar notificações enviadas pelo servidor (ex: logs)
		// Por enquanto ignoramos ou poderíamos logar no stderr
	}
}
