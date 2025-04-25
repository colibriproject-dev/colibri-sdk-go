package security

import (
	"context"

	"firebase.google.com/go/auth"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/cloud"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

type firebaseAuthService struct {
	client *auth.Client
}

func newFirebaseAuthenticationService() *firebaseAuthService {
	auth, err := cloud.GetFirebaseSession().Auth(context.Background())
	if err != nil {
		logging.Fatal(context.Background()).Msgf(connection_error, err)
	}

	return &firebaseAuthService{auth}
}

func (s *firebaseAuthService) GetUser(ctx context.Context, id string) (*User, error) {
	user, err := s.client.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:       user.UID,
		Email:    user.Email,
		Phone:    user.PhoneNumber,
		Name:     user.DisplayName,
		TenantID: user.CustomClaims[tenantIdField].(string),
		Profile:  user.CustomClaims[profileField].(string),
		Profiles: user.CustomClaims[profilesField].([]string),
		PhotoURL: user.PhotoURL,
	}, nil
}

func (s *firebaseAuthService) CreateUser(ctx context.Context, user *UserCreate) error {
	userToCreate := (&auth.UserToCreate{}).
		UID(user.ID).
		DisplayName(user.Name)

	if user.Email != "" {
		userToCreate.Email(user.Email)
	}

	if user.Phone != "" {
		userToCreate.PhoneNumber(user.Phone)
	}

	if user.Password != "" {
		userToCreate.Password(user.Password)
	}

	if user.PhotoURL != "" {
		userToCreate.PhotoURL(user.PhotoURL)
	}

	if _, err := s.client.CreateUser(ctx, userToCreate); err != nil {
		return err
	}

	userToSetClaims := &auth.UserToUpdate{}
	userToSetClaims.CustomClaims(map[string]any{
		profileField:  user.Profile,
		profilesField: user.Profiles,
		tenantIdField: user.TenantID,
	})
	_, err := s.client.UpdateUser(ctx, user.ID, userToSetClaims)

	return err
}

func (s *firebaseAuthService) UpdateUser(ctx context.Context, id string, user *UserUpdate) error {
	userToUpdate := &auth.UserToUpdate{}
	if user.Email != "" {
		userToUpdate.Email(user.Email)
	}

	if user.Phone != "" {
		userToUpdate.PhoneNumber(user.Phone)
	}

	if user.Password != "" {
		userToUpdate.Password(user.Password)
	}

	if user.Name != "" {
		userToUpdate.DisplayName(user.Name)
	}

	if user.PhotoURL != "" {
		userToUpdate.PhotoURL(user.PhotoURL)
	}

	userToUpdate.CustomClaims(map[string]any{
		profileField:  user.Profile,
		profilesField: user.Profiles,
		tenantIdField: user.TenantID,
	})

	_, err := s.client.UpdateUser(ctx, id, userToUpdate)
	return err
}

func (s *firebaseAuthService) DeleteUser(ctx context.Context, id string) error {
	return s.client.DeleteUser(ctx, id)
}

func (s *firebaseAuthService) EnableUser(ctx context.Context, id string) error {
	userToUpdate := (&auth.UserToUpdate{}).Disabled(false)
	_, err := s.client.UpdateUser(ctx, id, userToUpdate)
	return err
}

func (s *firebaseAuthService) DisableUser(ctx context.Context, id string) error {
	userToUpdate := (&auth.UserToUpdate{}).Disabled(true)
	_, err := s.client.UpdateUser(ctx, id, userToUpdate)
	return err
}
