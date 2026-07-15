package messaging

import (
	"context"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/security"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type TestMessage struct {
	Field string `json:"field" validate:"required"`
}

func TestNewProviderMessage(t *testing.T) {
	test.InitializeBaseTest()

	t.Run("should create a new provider message with correct fields when context and message are provided", func(t *testing.T) {
		authContext := security.NewAuthenticationContext("test_tenant_id", "test_user_id")
		ctx := authContext.SetInContext(context.Background())
		action := "test_action"
		message := "test_message"

		providerMessage := NewProviderMessage(ctx, action, message)

		assert.NotEqual(t, uuid.Nil, providerMessage.ID)
		assert.EqualValues(t, config.APP_NAME, providerMessage.Origin)
		assert.EqualValues(t, action, providerMessage.Action)
		assert.EqualValues(t, message, providerMessage.Message)
		assert.EqualValues(t, authContext, providerMessage.AuthContext)
	})
}

func TestProviderMessage_String(t *testing.T) {
	t.Run("should return json string representation when message is marshalled", func(t *testing.T) {
		ctx := context.Background()
		message := TestMessage{Field: "test"}
		providerMessage := NewProviderMessage(ctx, "test", message)

		result := providerMessage.String()

		assert.Contains(t, result, `"field":"test"`)
		assert.Contains(t, result, `"action":"test"`)
	})
}

func TestProviderMessage_DecodeAndValidateMessage(t *testing.T) {
	test.InitializeBaseTest()

	t.Run("should decode and validate message successfully when valid message is provided", func(t *testing.T) {
		ctx := context.Background()
		message := TestMessage{Field: "test"}
		providerMessage := NewProviderMessage(ctx, "test", message)
		var decodedMessage TestMessage

		err := providerMessage.DecodeAndValidateMessage(&decodedMessage)

		assert.NoError(t, err)
		assert.Equal(t, message.Field, decodedMessage.Field)
	})

	t.Run("should return error when message validation fails", func(t *testing.T) {
		ctx := context.Background()
		message := TestMessage{} // Empty field should fail validation
		providerMessage := NewProviderMessage(ctx, "test", message)
		var decodedMessage TestMessage

		err := providerMessage.DecodeAndValidateMessage(&decodedMessage)

		assert.Error(t, err)
	})
}

func TestProviderMessage_DecodeMessage(t *testing.T) {
	test.InitializeBaseTest()

	t.Run("should decode message successfully when valid message is provided", func(t *testing.T) {
		ctx := context.Background()
		message := TestMessage{Field: "test"}
		providerMessage := NewProviderMessage(ctx, "test", message)
		var decodedMessage TestMessage

		err := providerMessage.DecodeMessage(&decodedMessage)

		assert.NoError(t, err)
		assert.Equal(t, message.Field, decodedMessage.Field)
	})

	t.Run("should return error when message is invalid", func(t *testing.T) {
		ctx := context.Background()
		providerMessage := NewProviderMessage(ctx, "test", make(chan int)) // Invalid message type
		var decodedMessage TestMessage

		err := providerMessage.DecodeMessage(&decodedMessage)

		assert.Error(t, err)
	})
}

func TestProviderMessage_addOriginBrokerNotification(t *testing.T) {
	t.Run("should add origin broker notification when notification is provided", func(t *testing.T) {
		ctx := context.Background()
		providerMessage := NewProviderMessage(ctx, "test", "test")
		notification := "test_notification"

		providerMessage.addOriginBrokerNotification(notification)

		assert.Equal(t, notification, providerMessage.n)
	})
}

func TestProviderMessage_ReceiptMetadata(t *testing.T) {
	t.Run("should return nil metadata when message was not consumed", func(t *testing.T) {
		providerMessage := NewProviderMessage(context.Background(), "test", "test")

		assert.Nil(t, providerMessage.Attributes())
		assert.Nil(t, providerMessage.DeliveryAttempt())
	})

	t.Run("should expose attributes and delivery attempt after they are set on consumption", func(t *testing.T) {
		providerMessage := NewProviderMessage(context.Background(), "test", "test")
		attempt := 3

		providerMessage.setReceiptMetadata(map[string]string{"CloudPubSubDeadLetterSourceDeliveryCount": "5"}, &attempt)

		assert.Equal(t, "5", providerMessage.Attributes()["CloudPubSubDeadLetterSourceDeliveryCount"])
		assert.Equal(t, 3, *providerMessage.DeliveryAttempt())
	})

	t.Run("should not serialize receipt metadata into the published envelope", func(t *testing.T) {
		attempt := 7
		providerMessage := NewProviderMessage(context.Background(), "test", TestMessage{Field: "value"})
		providerMessage.setReceiptMetadata(map[string]string{"some-attr": "x"}, &attempt)

		result := providerMessage.String()

		assert.NotContains(t, result, "some-attr")
		assert.NotContains(t, result, "deliveryAttempt")
	})
}
