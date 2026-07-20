package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const minimumCredentialFailureDuration = time.Second

// LoginParams contains a local-login attempt.
type LoginParams struct {
	Email    string
	Password string
}

// Login checks local credentials and starts a session.
func (s *Service) Login(ctx context.Context, params LoginParams) (*directory.User, error) {
	started := time.Now()
	email := strings.TrimSpace(params.Email)
	user, passwordHash, err := s.passwordLoginCandidate(ctx, email)
	if err != nil {
		return nil, err
	}

	valid, err := directory.VerifyPassword(params.Password, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if user == nil || !valid {
		// Only credential failures are padded; successful and internal-error paths
		// return as soon as their work completes.
		time.Sleep(time.Until(started.Add(minimumCredentialFailureDuration)))
		return nil, ErrInvalidCredentials
	}

	if err := s.startSession(ctx, user.ID); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) passwordLoginCandidate(
	ctx context.Context,
	email string,
) (*directory.User, string, error) {
	user, err := s.users.GetLoginByEmail(ctx, email)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, s.dummyHash, nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get user: %w", err)
	}
	return user, user.PasswordHash, nil
}
