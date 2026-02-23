package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SSEClientTransport implementa a interface Transport usando Server-Sent Events.
type SSEClientTransport struct {
	baseURL     string
	httpClient  *http.Client
	msgEndpoint string
	headers     http.Header

	receiveChan chan Message
	errChan     chan error

	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	isClosed bool

	reconnectDelay time.Duration
}

const (
	maxReconnectDelay     = 30 * time.Second
	initialReconnectDelay = 1 * time.Second
)

// NewSSEClientTransport cria uma nova instância de transporte SSE para o cliente.
func NewSSEClientTransport(baseURL string, headers http.Header) *SSEClientTransport {
	if headers == nil {
		headers = make(http.Header)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SSEClientTransport{
		baseURL:     baseURL,
		httpClient:  &http.Client{},
		headers:     headers,
		receiveChan: make(chan Message, 100),
		errChan:     make(chan error, 1),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Connect estabelece a conexão SSE e inicia a escuta do stream.
func (t *SSEClientTransport) Connect(ctx context.Context) error {
	// Vincula o contexto externo ao ciclo de vida do transporte
	go func() {
		select {
		case <-ctx.Done():
			t.Close()
		case <-t.ctx.Done():
		}
	}()
	return t.connectWithRetry()
}

func (t *SSEClientTransport) connectWithRetry() error {
	t.mu.Lock()
	if t.isClosed {
		t.mu.Unlock()
		return io.EOF
	}
	t.mu.Unlock()

	req, err := http.NewRequestWithContext(t.ctx, "GET", t.baseURL, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição SSE: %w", err)
	}

	for k, v := range t.headers {
		req.Header[k] = v
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return t.handleReconnect(fmt.Errorf("erro ao conectar ao stream SSE: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return t.handleReconnect(fmt.Errorf("status HTTP inesperado ao conectar SSE: %d", resp.StatusCode))
	}

	// Reset delay ao conectar com sucesso
	t.mu.Lock()
	t.reconnectDelay = initialReconnectDelay
	t.mu.Unlock()

	go t.readStream(resp.Body)

	return nil
}

func (t *SSEClientTransport) handleReconnect(err error) error {
	t.mu.Lock()
	if t.isClosed {
		t.mu.Unlock()
		return io.EOF
	}

	delay := t.reconnectDelay
	if delay == 0 {
		delay = initialReconnectDelay
	}
	t.reconnectDelay = delay * 2
	if t.reconnectDelay > maxReconnectDelay {
		t.reconnectDelay = maxReconnectDelay
	}
	t.mu.Unlock()

	log.Printf("[MCP] Conexão SSE perdida, tentando reconectar em %v: %v", delay, err)

	select {
	case <-t.ctx.Done():
		return t.ctx.Err()
	case <-time.After(delay):
		return t.connectWithRetry()
	}
}

func (t *SSEClientTransport) readStream(body io.ReadCloser) {
	defer body.Close()
	scanner := bufio.NewScanner(body)
	var dataBuffer strings.Builder

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			if !scanner.Scan() {
				err := scanner.Err()
				if err == nil {
					err = io.EOF
				}

				select {
				case <-t.ctx.Done():
					return
				default:
					_ = t.handleReconnect(err)
					return
				}
			}

			line := scanner.Text()
			if line == "" {
				// Fim de um evento SSE, processar dados acumulados
				if dataBuffer.Len() > 0 {
					t.processMessage(dataBuffer.String())
					dataBuffer.Reset()
				}
				continue
			}

			if strings.HasPrefix(line, "event: ") {
				eventType := strings.TrimPrefix(line, "event: ")
				if eventType == "endpoint" {
					if scanner.Scan() {
						dataLine := scanner.Text()
						if strings.HasPrefix(dataLine, "data: ") {
							endpoint := strings.TrimPrefix(dataLine, "data: ")
							t.resolveEndpoint(endpoint)
						}
					}
				}
				continue
			}

			if strings.HasPrefix(line, "data: ") {
				if dataBuffer.Len() > 0 {
					dataBuffer.WriteString("\n")
				}
				dataBuffer.WriteString(strings.TrimPrefix(line, "data: "))
			}
		}
	}
}

func (t *SSEClientTransport) resolveEndpoint(endpoint string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	u, err := url.Parse(endpoint)
	if err != nil {
		log.Printf("[MCP] Erro ao parsear endpoint recebido: %v", err)
		return
	}

	if u.IsAbs() {
		t.msgEndpoint = endpoint
	} else {
		base, err := url.Parse(t.baseURL)
		if err != nil {
			log.Printf("[MCP] Erro ao parsear baseURL: %v", err)
			return
		}
		t.msgEndpoint = base.ResolveReference(u).String()
	}
}

func (t *SSEClientTransport) processMessage(data string) {
	var msg Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		// Ignorar mensagens malformadas
		return
	}

	select {
	case t.receiveChan <- msg:
	case <-t.ctx.Done():
	}
}

// Send envia uma mensagem MCP via POST JSON para o endpoint de mensagens.
func (t *SSEClientTransport) Send(msg Message) error {
	t.mu.Lock()
	endpoint := t.msgEndpoint
	t.mu.Unlock()

	if endpoint == "" {
		// Se ainda não descobrimos o endpoint, aguardamos um pouco ou retornamos erro
		// No fluxo MCP, o endpoint deve vir antes da primeira necessidade de envio
		return fmt.Errorf("endpoint de mensagens ainda não descoberto via SSE")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	req, err := http.NewRequestWithContext(t.ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição POST: %w", err)
	}

	for k, v := range t.headers {
		req.Header[k] = v
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem via POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status HTTP inesperado ao enviar mensagem: %d", resp.StatusCode)
	}

	return nil
}

// Receive aguarda e retorna a próxima mensagem do transporte.
func (t *SSEClientTransport) Receive() (Message, error) {
	select {
	case msg := <-t.receiveChan:
		return msg, nil
	case err := <-t.errChan:
		return Message{}, err
	case <-t.ctx.Done():
		return Message{}, io.EOF
	}
}

// Close encerra o transporte e libera recursos.
func (t *SSEClientTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isClosed {
		return nil
	}

	t.isClosed = true
	t.cancel()
	return nil
}
