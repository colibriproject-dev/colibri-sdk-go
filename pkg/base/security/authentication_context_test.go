package security

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthenticationContext(t *testing.T) {
	var defaultUserId = uuid.MustParse("5e859dae-c879-11eb-b8bc-0242ac130003").String()
	var defaultTenantId = uuid.MustParse("5e859dae-c879-11eb-b8bc-0242ac130004").String()

	t.Run("Should return tenant and user", func(t *testing.T) {
		result := NewAuthenticationContext(defaultTenantId, defaultUserId)
		assert.NotNil(t, result)
		assert.Equal(t, defaultTenantId, result.GetTenantID())
		assert.Equal(t, defaultUserId, result.GetUserID())
	})

	t.Run("Should set in context", func(t *testing.T) {
		result := NewAuthenticationContext(defaultTenantId, defaultUserId).SetInContext(context.Background())
		assert.NotNil(t, result)
		assert.NotNil(t, result.Value(contextKeyAuthenticationContext))
	})

	t.Run("Should return nil when context is nil", func(t *testing.T) {
		result := GetAuthenticationContext(context.Background())
		assert.Nil(t, result)
	})

	t.Run("Should return formatted string with tenant and user IDs", func(t *testing.T) {
		authContext := NewAuthenticationContext(defaultTenantId, defaultUserId)
		expected := fmt.Sprintf("tenantId: %s | userId: %s", defaultTenantId, defaultUserId)
		assert.Equal(t, expected, authContext.String())
	})

	t.Run("Should get in context", func(t *testing.T) {
		context := NewAuthenticationContext(defaultTenantId, defaultUserId).SetInContext(context.Background())
		assert.NotNil(t, context)

		result := GetAuthenticationContext(context)
		assert.Equal(t, defaultTenantId, result.GetTenantID())
		assert.Equal(t, defaultUserId, result.GetUserID())
	})
}
