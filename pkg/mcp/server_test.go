package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type mockTransport struct {
	sent    []Message
	receive chan Message
	closed  bool
	mu      sync.Mutex
}

func (m *mockTransport) Send(msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockTransport) Receive() (Message, error) {
	msg, ok := <-m.receive
	if !ok {
		return Message{}, nil // ou io.EOF simulado
	}
	return msg, nil
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}

type TestArgs struct {
	Query string `json:"query" jsonschema:"description=Busca"`
}

func TestServer_RegisterAndCall(t *testing.T) {
	s := NewServer("test-server", "1.0.0")

	handlerCalled := false
	s.RegisterTool("test-tool", "Uma ferramenta de teste", TestArgs{}, func(ctx context.Context, params any) (any, error) {
		handlerCalled = true
		args := params.(*TestArgs)
		if args.Query != "hello" {
			t.Errorf("expected query 'hello', got %s", args.Query)
		}
		return map[string]string{"result": "world"}, nil
	})

	// Testa listagem
	resList := s.handleListTools()
	tools := resList["tools"].([]Tool)
	if len(tools) != 1 || tools[0].Name != "test-tool" {
		t.Errorf("ferramenta não registrada corretamente")
	}

	// Testa chamada
	callParams := json.RawMessage(`{"name":"test-tool","arguments":{"query":"hello"}}`)
	resCall, err := s.handleCallTool(context.Background(), callParams)
	if err != nil {
		t.Fatalf("erro ao chamar ferramenta: %v", err)
	}

	if !handlerCalled {
		t.Error("handler não foi chamado")
	}

	resultMap := resCall.(map[string]string)
	if resultMap["result"] != "world" {
		t.Errorf("resultado inesperado: %v", resultMap)
	}
}

func TestServer_Handshake(t *testing.T) {
	s := NewServer("test-server", "1.0.0")
	res := s.handleInitialize()

	if res["serverInfo"].(ServerInfo).Name != "test-server" {
		t.Error("nome do servidor incorreto no initialize")
	}

	capabilities := res["capabilities"].(ServerCapabilities)
	if capabilities.Tools == nil {
		t.Error("capabilities.Tools deve estar presente")
	}
}

func TestServer_PanicRecovery(t *testing.T) {
	s := NewServer("test-server", "1.0.0")

	s.RegisterTool("panic-tool", "Ferramenta que dá panic", TestArgs{}, func(ctx context.Context, params any) (any, error) {
		panic("algo deu errado")
	})

	callParams := json.RawMessage(`{"name":"panic-tool","arguments":{"query":"test"}}`)
	_, err := s.handleCallTool(context.Background(), callParams)

	if err == nil {
		t.Fatal("esperava erro ao ocorrer panic")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("esperava *RPCError, obteve %T", err)
	}

	if rpcErr.Code != -32603 {
		t.Errorf("esperava código -32603, obteve %d", rpcErr.Code)
	}
}

func TestServer_Stop(t *testing.T) {
	s := NewServer("test-server", "1.0.0")
	transport := &mockTransport{
		receive: make(chan Message),
	}

	errChan := make(chan error)
	go func() {
		errChan <- s.Serve(transport)
	}()

	s.Stop()

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Serve retornou erro após Stop: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Serve não parou após Stop")
	}
}

func TestServer_SendLog(t *testing.T) {
	s := NewServer("test-server", "1.0.0")
	transport := &mockTransport{
		receive: make(chan Message),
	}

	// Inicia o servidor para associar o transporte
	go s.Serve(transport)
	// Pequena pausa para garantir que o transporte foi associado
	time.Sleep(10 * time.Millisecond)

	err := s.SendLog(LogLevelInfo, "test message")
	if err != nil {
		t.Fatalf("SendLog falhou: %v", err)
	}

	if len(transport.sent) != 1 {
		t.Fatalf("esperava 1 mensagem enviada, obteve %d", len(transport.sent))
	}

	msg := transport.sent[0]
	if msg.Method != "notifications/message" {
		t.Errorf("método incorreto: %s", msg.Method)
	}

	var params map[string]any
	json.Unmarshal(msg.Params, &params)
	if params["level"] != "info" {
		t.Errorf("level incorreto: %v", params["level"])
	}
	if params["data"] != "test message" {
		t.Errorf("data incorreta: %v", params["data"])
	}

	s.Stop()
}
