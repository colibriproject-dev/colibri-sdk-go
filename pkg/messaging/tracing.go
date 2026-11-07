package messaging

import (
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
)

// OpenTelemetry messaging semantic conventions. Backends name producer and consumer
// transactions from these attributes, not from the span name: without
// messaging.destination.name New Relic reports every consumer as
// "OtherTransaction/consumer/unknown".
const (
	attrMessagingSystem          = "messaging.system"
	attrMessagingDestinationName = "messaging.destination.name"
	attrMessagingOperationType   = "messaging.operation.type"
	attrMessagingOperationName   = "messaging.operation.name"
	attrMessagingMessageID       = "messaging.message.id"

	operationTypeSend    = "send"
	operationTypeProcess = "process"
	operationNamePublish = "publish"
	operationNameProcess = "process"

	systemRabbitMQ  = "rabbitmq"
	systemAWSSNS    = "aws.sns"
	systemAWSSQS    = "aws_sqs"
	systemGCPPubSub = "gcp_pubsub"
)

// brokerSystems returns the messaging.system of the configured broker on each side. AWS
// publishes to SNS and consumes from SQS, so the two sides differ there. It mirrors
// newProvider and returns empty values when no supported broker is configured.
func brokerSystems() (publish, consume string) {
	if config.COLIBRI_MESSAGING == config.MESSAGING_RABBITMQ {
		return systemRabbitMQ, systemRabbitMQ
	}

	switch config.CLOUD {
	case config.CLOUD_AWS:
		return systemAWSSNS, systemAWSSQS
	case config.CLOUD_GCP, config.CLOUD_FIREBASE:
		return systemGCPPubSub, systemGCPPubSub
	}

	return "", ""
}

// addMessagingAttributes describes a producer or consumer transaction with the messaging
// semantic conventions. An unknown system is left out rather than reported as empty.
func addMessagingAttributes(txn any, system, destination, operationType, operationName, messageID string) {
	if system != "" {
		monitoring.AddTransactionAttribute(txn, attrMessagingSystem, system)
	}
	monitoring.AddTransactionAttribute(txn, attrMessagingDestinationName, destination)
	monitoring.AddTransactionAttribute(txn, attrMessagingOperationType, operationType)
	monitoring.AddTransactionAttribute(txn, attrMessagingOperationName, operationName)
	monitoring.AddTransactionAttribute(txn, attrMessagingMessageID, messageID)
}
