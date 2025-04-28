package security

import (
	"context"
	"fmt"
)

// IAuthenticationContext is the external user contract
type IAuthenticationContext interface {
	GetUserID() string
	GetTenantID() string
}

// AuthenticationContext is the external user struct
type AuthenticationContext struct {
	TenantID string `json:"tenantId"`
	UserID   string `json:"userId"`
}

// NewAuthenticationContext returns a pointer of authentication context
func NewAuthenticationContext(tenantID, userID string) *AuthenticationContext {
	return &AuthenticationContext{TenantID: tenantID, UserID: userID}
}

// GetTenantID returns the tenant id
func (a *AuthenticationContext) GetTenantID() string {
	return a.TenantID
}

// GetUserID returns the user id
func (a *AuthenticationContext) GetUserID() string {
	return a.UserID
}

// SetInContext returns a context with authentication inside
func (a *AuthenticationContext) SetInContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeyAuthenticationContext, a)
}

// String returns the details of auth context
func (a *AuthenticationContext) String() string {
	return fmt.Sprintf("tenantId: %s | userId: %s", a.TenantID, a.UserID)
}

// GetAuthenticationContext return the authentication context inside the context
func GetAuthenticationContext(ctx context.Context) *AuthenticationContext {
	if result := ctx.Value(contextKeyAuthenticationContext); result != nil {
		return result.(*AuthenticationContext)
	}
	return nil
}

// Valid returns a boolean if the context is valid
func (a *AuthenticationContext) Valid() bool {
	return a.TenantID != "" && a.UserID != ""
}
