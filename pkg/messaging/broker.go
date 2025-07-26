package messaging

// OriginalMessage defines the interface for acknowledging or rejecting a message from a message broker.
type OriginalMessage interface {
	// Ack acknowledges the message.
	Ack() error
	// Nack rejects the message.
	// If requeue is true, the message will be put back in the original queue.
	// If requeue is false, the message will be discarded or sent to a DLQ.
	Nack(requeue bool, err error) error
}
