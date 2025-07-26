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
	ID          uuid.UUID                       `json:"id"`
	Origin      string                          `json:"origin"`
	Action      string                          `json:"action"`
	Message     any                             `json:"message"`
	AuthContext *security.AuthenticationContext `json:"authenticationContext"`
	n           any
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

// Ack acknowledges the message.
func (msg *ProviderMessage) Ack() error {
	if originalMessage, ok := msg.n.(OriginalMessage); ok {
		return originalMessage.Ack()
	}
	return nil
}

// Nack rejects the message.
// If requeue is true, the message will be put back in the original queue.
// If requeue is false, the message will be discarded or sent to a DLQ.
func (msg *ProviderMessage) Nack(requeue bool, err error) error {
	if originalMessage, ok := msg.n.(OriginalMessage); ok {
		return originalMessage.Nack(requeue, err)
	}
	return nil
}
