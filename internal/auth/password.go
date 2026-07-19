package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

// LoginParams contains a local-login attempt and its resolved client address.
type LoginParams struct {
	ClientIP string
	Email    string
	Password string
}

// Login checks local credentials and starts a session. The configured initial
// administrator owns its email on this login path and never falls through to a
// directory account with the same email.
func (s *Service) Login(ctx context.Context, params LoginParams) (*Principal, error) {
	email := strings.ToLower(strings.TrimSpace(params.Email))
	key := loginAttemptKey{clientIP: params.ClientIP, email: email}
	if !s.loginLimiter.allow(key) {
		return nil, ErrTooManyAttempts
	}

	if s.initialAdmin != nil && email == s.initialAdmin.email {
		valid, err := directory.VerifyPassword(params.Password, s.initialAdmin.passwordHash)
		if err != nil {
			return nil, fmt.Errorf("verify initial admin password: %w", err)
		}
		if !valid {
			return nil, ErrInvalidCredentials
		}
		if err := s.startInitialAdminSession(ctx); err != nil {
			return nil, err
		}
		s.loginLimiter.reset(key)
		return s.initialAdmin.principal(), nil
	}

	user, err := s.users.GetLoginByEmail(ctx, email)
	if errors.Is(err, dbutil.ErrNotFound) {
		if _, verifyErr := directory.VerifyPassword(params.Password, s.dummyHash); verifyErr != nil {
			return nil, fmt.Errorf("verify dummy password: %w", verifyErr)
		}
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	valid, err := directory.VerifyPassword(params.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return nil, ErrInvalidCredentials
	}

	if err := s.startPersistedSession(ctx, user.ID); err != nil {
		return nil, err
	}
	s.loginLimiter.reset(key)
	return principalFromUser(user), nil
}
