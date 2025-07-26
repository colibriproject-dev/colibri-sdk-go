package messaging

import "context"

type QueueConfiguration struct {
	topicName string
}

// QueueConsumer defines an interface for consuming messages from a queue and managing the queue's name.
type QueueConsumer interface {

	// Consume processes a ProviderMessage in a given context and returns an error if the processing fails.
	Consume(ctx context.Context, providerMessage *ProviderMessage) error

	// QueueName retrieves the name of the queue associated with the consumer.
	QueueName() string
}

// QueueConsumerConfig combines queue consumption capabilities with access to queue configuration details.
type QueueConsumerConfig interface {
	QueueConsumer

	// Config returns the queue configuration details by providing access to a QueueConfiguration object.
	Config() *QueueConfiguration
}
