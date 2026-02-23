package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// StdioTransport implementa a interface Transport utilizando os fluxos padrão do sistema (stdin/stdout).
//
// ATENÇÃO: O os.Stdout é de uso exclusivo do protocolo MCP.
// Qualquer log de depuração da aplicação DEVE ser direcionado para o os.Stderr.
// Caso contrário, o cliente MCP (como o Claude Desktop) falhará ao tentar interpretar logs como mensagens do protocolo.
type StdioTransport struct {
	scanner *bufio.Scanner
	stdout  io.Writer
	mu      sync.Mutex
}

// NewStdioTransport cria uma nova instância de StdioTransport.
func NewStdioTransport() *StdioTransport {
	scanner := bufio.NewScanner(os.Stdin)
	// Configura o buffer para suportar payloads de até 1MB.
	scanner.Buffer(nil, 1024*1024)

	return &StdioTransport{
		scanner: scanner,
		stdout:  os.Stdout,
	}
}

// Send serializa a mensagem em JSON e a envia para o stdout seguida de uma nova linha.
func (t *StdioTransport) Send(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	_, err = fmt.Fprintln(t.stdout, string(data))
	if err != nil {
		return fmt.Errorf("erro ao escrever no stdout: %w", err)
	}

	return nil
}

// Receive lê a próxima linha do stdin e a deserializa em uma Message.
func (t *StdioTransport) Receive() (Message, error) {
	if !t.scanner.Scan() {
		err := t.scanner.Err()
		if err == nil {
			return Message{}, io.EOF
		}
		return Message{}, fmt.Errorf("erro ao ler do stdin: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(t.scanner.Bytes(), &msg); err != nil {
		return Message{}, fmt.Errorf("erro ao deserializar mensagem: %w", err)
	}

	return msg, nil
}

// Close fecha o transporte. Para StdioTransport, não há recursos extras para fechar
// além dos fluxos padrão que são gerenciados pelo sistema operacional.
func (t *StdioTransport) Close() error {
	return nil
}
