package mcp

// Transport define a interface genérica para comunicação no protocolo MCP.
type Transport interface {
	// Send envia uma mensagem através do transporte.
	Send(msg Message) error
	// Receive aguarda e retorna a próxima mensagem do transporte.
	// Deve retornar io.EOF quando o transporte for fechado.
	Receive() (Message, error)
	// Close encerra o transporte e libera recursos.
	Close() error
}
