package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSEClientTransport(t *testing.T) {
	// 1. Criar um servidor mock para SSE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)

			// Enviar endpoint event
			fmt.Fprint(w, "event: endpoint\ndata: /messages\n\n")
			w.(http.Flusher).Flush()

			// Enviar uma mensagem de teste
			msg := Message{
				JSONRPC: "2.0",
				Method:  "test/method",
				Params:  json.RawMessage(`{"foo":"bar"}`),
			}
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			w.(http.Flusher).Flush()

			// Manter a conexão aberta até o cliente fechar ou timeout do teste
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
			return
		}

		if r.Method == "POST" && r.URL.Path == "/messages" {
			var msg Message
			if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if msg.Method == "ping" {
				w.WriteHeader(http.StatusAccepted)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// 2. Testar o transporte
	transport := NewSSEClientTransport(server.URL, nil)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("falha ao conectar: %v", err)
	}

	// 3. Verificar recebimento
	msg, err := transport.Receive()
	if err != nil {
		t.Fatalf("erro ao receber: %v", err)
	}

	if msg.Method != "test/method" {
		t.Errorf("método esperado 'test/method', obtido %s", msg.Method)
	}

	// Pequena pausa para garantir que o endpoint foi processado
	time.Sleep(100 * time.Millisecond)

	// 4. Verificar envio
	pingMsg := Message{
		JSONRPC: "2.0",
		Method:  "ping",
	}
	if err := transport.Send(pingMsg); err != nil {
		t.Errorf("falha ao enviar: %v", err)
	}
}

func TestSSEClientTransport_EndpointResolution(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		data     string
		expected string
	}{
		{
			name:     "Relative path",
			baseURL:  "http://localhost:8080/sse",
			data:     "/messages",
			expected: "http://localhost:8080/messages",
		},
		{
			name:     "Absolute URL",
			baseURL:  "http://localhost:8080/sse",
			data:     "http://remote:9090/msg",
			expected: "http://remote:9090/msg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewSSEClientTransport(tt.baseURL, nil)
			tr.resolveEndpoint(tt.data)
			actual := tr.msgEndpoint

			if actual != tt.expected {
				t.Errorf("esperado %s, obtido %s", tt.expected, actual)
			}
		})
	}
}

func TestSSEClientTransport_MultiLineData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		fmt.Fprint(w, "data: {\n")
		fmt.Fprint(w, "data: \"jsonrpc\": \"2.0\",\n")
		fmt.Fprint(w, "data: \"method\": \"multi/line\"\n")
		fmt.Fprint(w, "data: }\n\n")
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	tr := NewSSEClientTransport(server.URL, nil)
	defer tr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := tr.Connect(ctx); err != nil {
		t.Fatalf("falha ao conectar: %v", err)
	}

	msg, err := tr.Receive()
	if err != nil {
		t.Fatalf("erro ao receber: %v", err)
	}

	if msg.Method != "multi/line" {
		t.Errorf("método esperado 'multi/line', obtido %s", msg.Method)
	}
}

func TestSSEClientTransport_Reconnect(t *testing.T) {
	connectionCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectionCount++
		if connectionCount == 1 {
			// Fecha a conexão imediatamente no primeiro acesso
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {\"jsonrpc\": \"2.0\", \"method\": \"reconnected\"}\n\n")
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	// Reduzir o delay inicial para o teste não demorar
	tr := NewSSEClientTransport(server.URL, nil)
	defer tr.Close()

	// Injetar delay pequeno manualmente (como não exportamos o campo, poderíamos exportar ou usar reflexão, mas vamos confiar no padrão)
	// Para este teste, vamos apenas aguardar a reconexão.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// O primeiro Connect deve falhar ou iniciar a reconexão em background.
	// Na nossa implementação, Connect retorna o erro do handleReconnect que aguarda o delay.
	// Então o primeiro Connect vai demorar pelo menos 1s se falhar.

	if err := tr.Connect(ctx); err != nil {
		t.Fatalf("Connect falhou mesmo com retry: %v", err)
	}

	msg, err := tr.Receive()
	if err != nil {
		t.Fatalf("erro ao receber após reconexão: %v", err)
	}

	if msg.Method != "reconnected" {
		t.Errorf("esperava 'reconnected', obtido %s", msg.Method)
	}

	if connectionCount < 2 {
		t.Errorf("esperava pelo menos 2 conexões ao servidor, obtido %d", connectionCount)
	}
}

func TestSSEClientTransport_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Manter aberto
		<-r.Context().Done()
	}))
	defer server.Close()

	tr := NewSSEClientTransport(server.URL, nil)

	ctx, cancel := context.WithCancel(context.Background())
	if err := tr.Connect(ctx); err != nil {
		t.Fatalf("falha ao conectar: %v", err)
	}

	// Cancelar o contexto e verificar se o transporte para
	cancel()

	_, err := tr.Receive()
	if err != io.EOF {
		t.Errorf("esperava io.EOF ao cancelar contexto, obtido %v", err)
	}
}
