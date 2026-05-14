// Package auth owns local setup, login (password and SSO), API-key issuance,
// and session lookup for the Woodstar admin surface.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexedwards/scs/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/users"
)

const sessionUserKey = "user_id"

// Auth errors describe expected setup, login, and authentication failures.
var (
	ErrAlreadySetup       = errors.New("woodstar is already set up")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotAuthenticated   = errors.New("not authenticated")
	ErrNotSetup           = errors.New("woodstar setup is required")
)

// Service owns local setup, login, and session lookup behavior.
type Service struct {
	users    *users.Service
	sessions *scs.SessionManager
}

// NewService creates an auth service backed by a user service and an scs session manager.
func NewService(users *users.Service, sessions *scs.SessionManager) *Service {
	return &Service{users: users, sessions: sessions}
}

// CurrentUser returns the user attached to the session loaded into ctx by scs middleware.
func (s *Service) CurrentUser(ctx context.Context) (*users.User, error) {
	id := s.sessions.GetInt64(ctx, sessionUserKey)
	if id == 0 {
		return nil, ErrNotAuthenticated
	}
	user, err := s.users.Get(ctx, id)
	if errors.Is(err, dbutil.ErrNotFound) {
		_ = s.sessions.Destroy(ctx)
		return nil, ErrNotAuthenticated
	}
	if err != nil {
		return nil, fmt.Errorf("get session user: %w", err)
	}
	return user, nil
}

// Authenticate resolves the caller from either a Bearer API key in
// authHeader or, when authHeader is empty or malformed, the session cookie
// already loaded into ctx by scs middleware. Returns ErrNotAuthenticated for
// both missing and bad credentials.
func (s *Service) Authenticate(ctx context.Context, authHeader string) (*users.User, error) {
	if token, ok := bearerToken(authHeader); ok {
		return s.userByAPIKey(ctx, token)
	}
	return s.CurrentUser(ctx)
}

// Logout revokes the active session.
func (s *Service) Logout(ctx context.Context) error {
	if err := s.sessions.Destroy(ctx); err != nil {
		return fmt.Errorf("destroy session: %w", err)
	}
	return nil
}

// startSession rotates the session ID (CSRF defense on privilege change) and binds the user.
func (s *Service) startSession(ctx context.Context, userID int64) error {
	if err := s.sessions.RenewToken(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}
	s.sessions.Put(ctx, sessionUserKey, userID)
	return nil
}
