package messaging

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/security"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/validator"
	"github.com/google/uuid"
)

type ProviderMessage struct {
	ID              uuid.UUID                       `json:"id"`
	Origin          string                          `json:"origin"`
	Action          string                          `json:"action"`
	Message         any                             `json:"message"`
	AuthContext     *security.AuthenticationContext `json:"authenticationContext"`
	CorrelationID   string                          `json:"correlationId,omitempty"`
	n               any
	attributes      map[string]string
	deliveryAttempt *int
}

// NewProviderMessage returns a new ProviderMessage
func NewProviderMessage(ctx context.Context, action string, message any) *ProviderMessage {
	return &ProviderMessage{
		ID:          uuid.New(),
		Origin:      config.APP_NAME,
		Action:      action,
		Message:     message,
		AuthContext: security.GetAuthenticationContext(ctx),
	}
}

// NewConsumerMessage builds a ProviderMessage as if it had just been received from a
// broker. It is meant for testing QueueConsumer implementations without a live broker:
// message is what DecodeMessage/DecodeAndValidateMessage will yield, while attributes
// and deliveryAttempt are what Attributes() and DeliveryAttempt() return — letting a
// consumer test exercise dead-letter metadata such as
// CloudPubSubDeadLetterSourceDeliveryCount.
func NewConsumerMessage(action string, message any, attributes map[string]string, deliveryAttempt *int) *ProviderMessage {
	msg := &ProviderMessage{
		ID:      uuid.New(),
		Origin:  config.APP_NAME,
		Action:  action,
		Message: message,
	}
	msg.setReceiptMetadata(attributes, deliveryAttempt)
	return msg
}

// String convert struct into JSON string
func (msg *ProviderMessage) String() string {
	message, _ := json.Marshal(msg)

	return string(message)
}

// DecodeAndValidateMessage transform interface into ProviderMessage and validate the struct
func (msg *ProviderMessage) DecodeAndValidateMessage(model any) error {
	if err := msg.DecodeMessage(model); err != nil {
		return err
	}

	return validator.Struct(model)
}

// DecodeMessage transform interface into ProviderMessage
func (msg *ProviderMessage) DecodeMessage(model any) error {
	buf := new(bytes.Buffer)

	if err := json.NewEncoder(buf).Encode(msg.Message); err != nil {
		return err
	}

	if err := json.NewDecoder(buf).Decode(model); err != nil {
		return err
	}

	return nil
}

// addOriginBrokerNotification add reference of an origin broker message to send dlq if an error occurs
func (msg *ProviderMessage) addOriginBrokerNotification(n any) {
	msg.n = n
}

// setReceiptMetadata records the broker-level metadata observed when the message is
// consumed. Each provider calls it before delivering the message to the consumer, so
// consumers can read broker attributes and the delivery count in a uniform way.
func (msg *ProviderMessage) setReceiptMetadata(attributes map[string]string, deliveryAttempt *int) {
	msg.attributes = attributes
	msg.deliveryAttempt = deliveryAttempt
}

// Attributes returns the broker-level attributes/headers received with the message
// (GCP Pub/Sub message attributes, SQS message + system attributes, RabbitMQ headers).
// It is nil for messages that were just published and is populated only on consumption.
// Typical use: reading dead-letter metadata such as GCP's
// CloudPubSubDeadLetterSourceDeliveryCount on a dead-letter subscription.
func (msg *ProviderMessage) Attributes() map[string]string {
	return msg.attributes
}

// DeliveryAttempt returns the broker-reported delivery/receive count when the broker
// exposes it (GCP DeliveryAttempt, SQS ApproximateReceiveCount, RabbitMQ x-death count),
// or nil when unavailable.
func (msg *ProviderMessage) DeliveryAttempt() *int {
	return msg.deliveryAttempt
}

// Ack acknowledges the message.
func (msg *ProviderMessage) Ack(ctx context.Context) error {
	if originalMessage, ok := msg.n.(OriginalMessage); ok {
		return originalMessage.Ack(ctx)
	}
	return nil
}

// Nack rejects the message.
// If requeue is true, the message will be put back in the original queue.
// If requeue is false, the message will be discarded or sent to a DLQ.
func (msg *ProviderMessage) Nack(ctx context.Context, requeue bool, err error) error {
	if originalMessage, ok := msg.n.(OriginalMessage); ok {
		return originalMessage.Nack(ctx, requeue, err)
	}
	return nil
}
