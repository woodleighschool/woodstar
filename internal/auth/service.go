// Package auth handles login, API keys, and sessions.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexedwards/scs/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/httpx"
)

const sessionUserIDKey = "user_id"

// Auth errors for expected auth failures.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotAuthenticated   = errors.New("not authenticated")
)

// Service owns login and authenticated-user lookup.
type Service struct {
	users     *directory.UserService
	sessions  *scs.SessionManager
	dummyHash string
	oidc      *oidcProvider
}

// NewService wires authentication to users and sessions.
func NewService(users *directory.UserService, sessions *scs.SessionManager) (*Service, error) {
	dummyHash, err := directory.HashPassword("woodstar-dummy-password")
	if err != nil {
		return nil, fmt.Errorf("hash dummy password: %w", err)
	}

	return &Service{
		users:     users,
		sessions:  sessions,
		dummyHash: dummyHash,
	}, nil
}

// ConfigureOIDC performs OIDC issuer discovery and enables the SSO flow.
// Safe to call once at startup; subsequent calls overwrite the previous
// configuration. A discovery failure leaves SSO disabled.
func (s *Service) ConfigureOIDC(ctx context.Context, cfg OIDCConfig) error {
	provider, err := configureOIDC(ctx, cfg)
	if err != nil {
		return err
	}
	s.oidc = provider
	return nil
}

// CurrentUser resolves the persisted user attached to the session loaded into
// ctx by SCS middleware.
func (s *Service) CurrentUser(ctx context.Context) (*directory.User, error) {
	userID := s.sessions.GetInt64(ctx, sessionUserIDKey)
	if userID == 0 {
		return nil, ErrNotAuthenticated
	}

	user, err := s.users.Get(ctx, userID)
	if errors.Is(err, dbutil.ErrNotFound) {
		s.destroyInvalidSession(ctx)
		return nil, ErrNotAuthenticated
	}
	if err != nil {
		return nil, fmt.Errorf("get session user: %w", err)
	}
	if !user.CanLogin || user.Role == nil {
		s.destroyInvalidSession(ctx)
		return nil, ErrNotAuthenticated
	}
	return user, nil
}

// Authenticate resolves the caller from either a Bearer API key in
// authHeader or, when authHeader is empty or malformed, the session cookie
// already loaded into ctx by SCS middleware. Returns ErrNotAuthenticated for
// both missing and bad credentials.
func (s *Service) Authenticate(ctx context.Context, authHeader string) (*directory.User, error) {
	if token, ok := httpx.BearerToken(authHeader); ok {
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

func (s *Service) startSession(ctx context.Context, userID int64) error {
	if err := s.sessions.RenewToken(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}
	s.sessions.Put(ctx, sessionUserIDKey, userID)
	return nil
}

func (s *Service) destroyInvalidSession(ctx context.Context) {
	_ = s.sessions.Destroy(ctx)
}
