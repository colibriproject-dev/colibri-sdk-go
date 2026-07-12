package messaging

import "context"

// OriginalMessage defines the interface for acknowledging or rejecting a message from a message broker.
type OriginalMessage interface {
	// Ack acknowledges the message.
	Ack(ctx context.Context) error
	// Nack rejects the message.
	// If requeue is true, the message is put back in the original queue for immediate redelivery.
	// If requeue is false, the message is left to the broker's dead-letter handling:
	// RabbitMQ rejects it without requeue (routed to the configured DLX, or dropped if none);
	// SQS leaves it in-flight until the visibility timeout expires, after which the queue's
	// redrive policy moves it to the DLQ once maxReceiveCount is exceeded;
	// GCP Pub/Sub nacks it, so the subscription's retry policy redelivers it until the
	// dead-letter policy (if configured) forwards it to the dead-letter topic.
	Nack(ctx context.Context, requeue bool, err error) error
}
