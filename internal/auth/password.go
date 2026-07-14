package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

// SetupParams contains the first administrator account fields.
type SetupParams struct {
	Email    string
	Name     string
	Password string
}

// SetupComplete reports whether initial setup has completed.
func (s *Service) SetupComplete(ctx context.Context) (bool, error) {
	return s.users.SetupComplete(ctx)
}

// Setup creates the first administrator account and starts a session.
func (s *Service) Setup(ctx context.Context, params SetupParams) (*directory.User, error) {
	user, err := s.users.CreateInitialAdministrator(ctx, directory.UserCreate{
		Email:    params.Email,
		Name:     params.Name,
		Password: params.Password,
	})
	if errors.Is(err, directory.ErrSetupComplete) {
		return nil, ErrAlreadySetup
	}
	if err != nil {
		return nil, fmt.Errorf("create setup user: %w", err)
	}

	if err := s.startSession(ctx, user.ID); err != nil {
		return nil, err
	}
	return user, nil
}

// Login checks local credentials and starts a session.
func (s *Service) Login(ctx context.Context, email string, password string) (*directory.User, error) {
	email = strings.TrimSpace(email)
	complete, err := s.users.SetupComplete(ctx)
	if err != nil {
		return nil, fmt.Errorf("check setup state: %w", err)
	}
	if !complete {
		return nil, ErrNotSetup
	}

	user, err := s.users.GetLoginByEmail(ctx, email)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	valid, err := directory.VerifyPassword(password, user.PasswordHash)
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
