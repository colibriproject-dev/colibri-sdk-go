package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestClient_Connect(t *testing.T) {
	// Mock do transporte
	receiveChan := make(chan Message, 10)
	transport := &mockTransport{
		receive: receiveChan,
		sent:    []Message{},
	}

	client := NewClient(transport)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Goroutine para simular a resposta do servidor ao initialize
	go func() {
		// Aguarda o envio do initialize
		for {
			time.Sleep(10 * time.Millisecond)
			transport.mu.Lock()
			found := false
			var requestID any
			for _, m := range transport.sent {
				if m.Method == "initialize" {
					found = true
					requestID = m.ID
					break
				}
			}
			transport.mu.Unlock()
			if found {
				// Responde o initialize
				initResult := map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    ServerCapabilities{},
					"serverInfo":      ServerInfo{Name: "test-server", Version: "1.0.0"},
				}
				resultJSON, _ := json.Marshal(initResult)
				receiveChan <- Message{
					JSONRPC: "2.0",
					ID:      requestID,
					Result:  resultJSON,
				}
				return
			}
		}
	}()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect falhou: %v", err)
	}

	if client.serverInfo.Name != "test-server" {
		t.Errorf("esperava server name 'test-server', obteve '%s'", client.serverInfo.Name)
	}

	// Verifica se enviou a notificação notifications/initialized
	transport.mu.Lock()
	foundInitialized := false
	for _, m := range transport.sent {
		if m.Method == "notifications/initialized" {
			foundInitialized = true
			break
		}
	}
	transport.mu.Unlock()

	if !foundInitialized {
		t.Error("não enviou a notificação notifications/initialized")
	}

	client.Close()
}

func TestClient_ListTools(t *testing.T) {
	receiveChan := make(chan Message, 10)
	transport := &mockTransport{
		receive: receiveChan,
		sent:    []Message{},
	}

	client := NewClient(transport)
	// Simula que já está conectado (sem passar pelo Connect para simplificar)
	receiveCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.listen(receiveCtx)

	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			transport.mu.Lock()
			var reqID any
			found := false
			for _, m := range transport.sent {
				if m.Method == "tools/list" {
					reqID = m.ID
					found = true
					break
				}
			}
			transport.mu.Unlock()

			if found {
				result := map[string]any{
					"tools": []Tool{
						{Name: "test-tool", Description: "desc", InputSchema: InputSchema{Type: "object"}},
					},
				}
				resultJSON, _ := json.Marshal(result)
				receiveChan <- Message{JSONRPC: "2.0", ID: reqID, Result: resultJSON}
				return
			}
		}
	}()

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools falhou: %v", err)
	}

	if len(tools) != 1 || tools[0].Name != "test-tool" {
		t.Errorf("ferramentas incorretas: %v", tools)
	}
}

func TestClient_CallTool(t *testing.T) {
	receiveChan := make(chan Message, 10)
	transport := &mockTransport{
		receive: receiveChan,
		sent:    []Message{},
	}

	client := NewClient(transport)
	receiveCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.listen(receiveCtx)

	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			transport.mu.Lock()
			var reqID any
			found := false
			for _, m := range transport.sent {
				if m.Method == "tools/call" {
					reqID = m.ID
					found = true
					break
				}
			}
			transport.mu.Unlock()

			if found {
				result := map[string]any{"content": []map[string]any{{"type": "text", "text": "result-text"}}}
				resultJSON, _ := json.Marshal(result)
				receiveChan <- Message{JSONRPC: "2.0", ID: reqID, Result: resultJSON}
				return
			}
		}
	}()

	resp, err := client.CallTool(context.Background(), "test-tool", map[string]int{"a": 1})
	if err != nil {
		t.Fatalf("CallTool falhou: %v", err)
	}

	if resp.Result == nil {
		t.Fatal("resultado é nil")
	}

	var resMap map[string]any
	json.Unmarshal(resp.Result, &resMap)
	if resMap["content"] == nil {
		t.Errorf("resultado inesperado: %s", string(resp.Result))
	}
}

func TestClient_Timeout(t *testing.T) {
	transport := &mockTransport{
		receive: make(chan Message),
		sent:    []Message{},
	}

	client := NewClient(transport)
	receiveCtx, cancelListen := context.WithCancel(context.Background())
	defer cancelListen()
	go client.listen(receiveCtx)

	ctx, cancelCall := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelCall()

	_, err := client.Call(ctx, "timeout-method", nil)
	if err == nil {
		t.Fatal("esperava erro de timeout, mas foi nil")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("esperava context.DeadlineExceeded, obteve: %v", err)
	}
}
