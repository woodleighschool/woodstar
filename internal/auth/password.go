package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/users"
)

// SetupParams contains the first administrator account fields.
type SetupParams struct {
	Email    string
	Name     string
	Password string
}

// SetupComplete reports whether the initial administrator account exists.
func (s *Service) SetupComplete(ctx context.Context) (bool, error) {
	return s.users.Exists(ctx)
}

// Setup creates the first administrator account and starts a session.
func (s *Service) Setup(ctx context.Context, params SetupParams) (*users.User, error) {
	exists, err := s.users.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("check setup state: %w", err)
	}
	if exists {
		return nil, ErrAlreadySetup
	}

	user, err := s.users.Create(ctx, users.UserCreate{
		Email:    params.Email,
		Name:     params.Name,
		Password: params.Password,
		Role:     users.RoleAdmin,
	})
	if err != nil {
		return nil, fmt.Errorf("create setup user: %w", err)
	}

	if err := s.startSession(ctx, user.ID); err != nil {
		return nil, err
	}
	return user, nil
}

// Login checks local credentials and starts a session.
func (s *Service) Login(ctx context.Context, email string, password string) (*users.User, error) {
	exists, err := s.users.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("check setup state: %w", err)
	}
	if !exists {
		return nil, ErrNotSetup
	}

	user, err := s.users.GetLoginByEmail(ctx, email)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	valid, err := users.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return nil, ErrInvalidCredentials
	}

	if err := s.startSession(ctx, user.ID); err != nil {
		return nil, err
	}
	return user, nil
}
