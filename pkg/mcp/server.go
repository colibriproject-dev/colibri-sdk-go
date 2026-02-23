package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sync"
)

// Server representa um servidor MCP.
type Server struct {
	info       ServerInfo
	tools      map[string]Tool
	handlers   map[string]ToolHandler
	inputTypes map[string]reflect.Type
	mu         sync.RWMutex
	transport  Transport
	cancel     context.CancelFunc
	ctx        context.Context
}

// NewServer cria uma nova instância de um servidor MCP.
func NewServer(name, version string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		info: ServerInfo{
			Name:    name,
			Version: version,
		},
		tools:      make(map[string]Tool),
		handlers:   make(map[string]ToolHandler),
		inputTypes: make(map[string]reflect.Type),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// RegisterTool registra uma nova ferramenta no servidor.
// O parâmetro inputStruct deve ser uma instância da struct que define os argumentos da ferramenta.
func (s *Server) RegisterTool(name, description string, inputStruct any, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputSchema := reflectStructToSchema(inputStruct)

	s.tools[name] = Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
	s.handlers[name] = handler
	s.inputTypes[name] = reflect.TypeOf(inputStruct)
}

// Serve inicia o loop de processamento de mensagens do servidor.
func (s *Server) Serve(t Transport) error {
	s.mu.Lock()
	s.transport = t
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.transport = nil
		s.mu.Unlock()
		t.Close()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			msg, err := t.Receive()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				// Se o contexto foi cancelado, ignoramos o erro de leitura
				select {
				case <-s.ctx.Done():
					return nil
				default:
				}
				return fmt.Errorf("erro ao receber mensagem: %w", err)
			}

			go s.handleMessage(t, msg)
		}
	}
}

// Stop sinaliza para o servidor parar de processar mensagens e encerra o transporte.
func (s *Server) Stop() {
	s.cancel()
}

func (s *Server) handleMessage(t Transport, msg Message) {
	// Se for uma resposta, ignoramos (servidor MCP geralmente não espera respostas a menos que peça logs/progresso)
	if msg.Result != nil || msg.Error != nil {
		return
	}

	// Se for uma notificação (sem ID)
	if msg.ID == nil {
		s.handleNotification(msg)
		return
	}

	// Se for uma requisição
	s.handleRequest(t, msg)
}

func (s *Server) handleNotification(msg Message) {
	switch msg.Method {
	case "notifications/initialized":
		fmt.Fprintf(os.Stderr, "Cliente MCP inicializado\n")
	default:
		fmt.Fprintf(os.Stderr, "Notificação não tratada: %s\n", msg.Method)
	}
}

func (s *Server) handleRequest(t Transport, msg Message) {
	var result any
	var err error

	switch msg.Method {
	case "initialize":
		result = s.handleInitialize()
	case "tools/list":
		result = s.handleListTools()
	case "tools/call":
		result, err = s.handleCallTool(s.ctx, msg.Params)
	default:
		err = &RPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Método não encontrado: %s", msg.Method),
		}
	}

	s.sendResponse(t, msg.ID, result, err)
}

func (s *Server) handleInitialize() map[string]any {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: false},
		},
		"serverInfo": s.info,
	}
}

func (s *Server) handleListTools() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}

	return map[string]any{
		"tools": tools,
	}
}

func (s *Server) handleCallTool(ctx context.Context, params json.RawMessage) (any, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: fmt.Sprintf("Argumentos inválidos: %v", err),
		}
	}

	s.mu.RLock()
	handler, exists := s.handlers[call.Name]
	inputType := s.inputTypes[call.Name]
	s.mu.RUnlock()

	if !exists {
		return nil, &RPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Ferramenta não encontrada: %s", call.Name),
		}
	}

	// Unmarshal dinâmico com Reflection
	var input any
	if inputType != nil {
		// Se inputType for um ponteiro, queremos o tipo base para reflect.New
		t := inputType
		isPtr := t.Kind() == reflect.Ptr
		if isPtr {
			t = t.Elem()
		}

		instance := reflect.New(t).Interface()
		if err := json.Unmarshal(call.Arguments, &instance); err != nil {
			return nil, &RPCError{
				Code:    -32602,
				Message: fmt.Sprintf("Parâmetros inválidos para a ferramenta %s: %v", call.Name, err),
			}
		}
		input = instance
	} else {
		input = call.Arguments
	}

	// Execução do Handler com captura de Panics
	return s.executeHandler(ctx, handler, input)
}

func (s *Server) executeHandler(ctx context.Context, handler ToolHandler, input any) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("Erro interno ao executar ferramenta (panic): %v", r)
			_ = s.SendLog(LogLevelError, panicMsg)

			err = &RPCError{
				Code:    -32603,
				Message: panicMsg,
			}
		}
	}()

	return handler(ctx, input)
}

func (s *Server) sendResponse(t Transport, id any, result any, rpcErr error) {
	resp := Message{
		JSONRPC: "2.0",
		ID:      id,
	}

	if rpcErr != nil {
		if e, ok := rpcErr.(*RPCError); ok {
			resp.Error = e
		} else {
			resp.Error = &RPCError{
				Code:    -32603,
				Message: rpcErr.Error(),
			}
		}
	} else {
		resBytes, _ := json.Marshal(result)
		resp.Result = resBytes
	}

	if err := t.Send(resp); err != nil {
		log.Printf("Erro ao enviar resposta: %v", err)
	}
}

// SendLog envia uma mensagem de log para o cliente através do transporte ativo.
func (s *Server) SendLog(level LogLevel, message string) error {
	s.mu.RLock()
	t := s.transport
	s.mu.RUnlock()

	if t == nil {
		return fmt.Errorf("transporte não disponível")
	}

	params := map[string]any{
		"level": level,
		"data":  message,
	}
	paramsBytes, _ := json.Marshal(params)

	notification := Message{
		JSONRPC: "2.0",
		Method:  "notifications/message",
		Params:  paramsBytes,
	}

	return t.Send(notification)
}

// SetAsDefaultLogger integra o servidor MCP com o sistema de logging do SDK,
// redirecionando logs para o cliente MCP.
func (s *Server) SetAsDefaultLogger() {
	// Nota: Como o pacote logging do SDK usa slog global e variáveis internas,
	// e o SDK Colibri parece ser baseado em slog, poderíamos implementar um slog.Handler
	// que chama s.SendLog.
}
