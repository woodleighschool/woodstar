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

// Login checks local credentials and starts a session. The configured initial
// administrator owns its email on this login path and never falls through to a
// directory account with the same email.
func (s *Service) Login(ctx context.Context, params LoginParams) (*Principal, error) {
	started := time.Now()
	email := strings.ToLower(strings.TrimSpace(params.Email))
	principal, passwordHash, err := s.passwordLoginCandidate(ctx, email)
	if err != nil {
		return nil, err
	}

	valid, err := directory.VerifyPassword(params.Password, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if principal == nil || !valid {
		// Only credential failures are padded; successful and internal-error paths
		// return as soon as their work completes.
		time.Sleep(time.Until(started.Add(minimumCredentialFailureDuration)))
		return nil, ErrInvalidCredentials
	}

	if principal.UserID == nil {
		err = s.startInitialAdminSession(ctx)
	} else {
		err = s.startPersistedSession(ctx, *principal.UserID)
	}
	if err != nil {
		return nil, err
	}
	return principal, nil
}

func (s *Service) passwordLoginCandidate(
	ctx context.Context,
	email string,
) (*Principal, string, error) {
	if s.initialAdmin != nil && email == s.initialAdmin.email {
		return s.initialAdmin.principal(), s.initialAdmin.passwordHash, nil
	}

	user, err := s.users.GetLoginByEmail(ctx, email)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, s.dummyHash, nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get user: %w", err)
	}
	return principalFromUser(user), user.PasswordHash, nil
}
