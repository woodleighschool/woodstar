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

const (
	sessionPrincipalKindKey = "principal_kind"
	sessionUserIDKey        = "user_id"

	principalKindPersistedUser = "persisted_user"
	principalKindInitialAdmin  = "initial_admin"
)

// Auth errors for expected auth failures.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotAuthenticated   = errors.New("not authenticated")
	ErrTooManyAttempts    = errors.New("too many login attempts")
)

// Principal is an authenticated Woodstar identity. UserID is nil for the
// deployment-configured initial administrator, which has no directory row.
type Principal struct {
	UserID *int64         `json:"id,omitempty"`
	Email  string         `json:"email" format:"email"`
	Name   string         `json:"name"`
	Role   directory.Role `json:"role"`
}

// InitialAdminConfig contains the optional deployment-controlled administrator.
// Its zero value disables that principal at the service boundary.
type InitialAdminConfig struct {
	Email    string
	Password string
}

type initialAdmin struct {
	email        string
	passwordHash string
}

// Service owns login and authenticated-principal lookup.
type Service struct {
	users        *directory.UserService
	sessions     *scs.SessionManager
	initialAdmin *initialAdmin
	loginLimiter *loginLimiter
	dummyHash    string
	oidc         *oidcProvider
}

// NewService wires authentication to users, sessions, and the optional
// deployment-configured administrator.
func NewService(
	users *directory.UserService,
	sessions *scs.SessionManager,
	initialConfig InitialAdminConfig,
) (*Service, error) {
	configured, err := newInitialAdmin(initialConfig)
	if err != nil {
		return nil, err
	}

	var dummyHash string
	if configured != nil {
		dummyHash = configured.passwordHash
	} else {
		dummyHash, err = directory.HashPassword("woodstar-dummy-password")
		if err != nil {
			return nil, fmt.Errorf("hash dummy password: %w", err)
		}
	}

	return &Service{
		users:        users,
		sessions:     sessions,
		initialAdmin: configured,
		loginLimiter: newLoginLimiter(),
		dummyHash:    dummyHash,
	}, nil
}

func newInitialAdmin(cfg InitialAdminConfig) (*initialAdmin, error) {
	email := cfg.Email
	if email == "" && cfg.Password == "" {
		return nil, nil
	}
	if email == "" || cfg.Password == "" {
		return nil, errors.New("initial admin email and password must both be set")
	}
	hash, err := directory.HashPassword(cfg.Password)
	if err != nil {
		return nil, fmt.Errorf("hash initial admin password: %w", err)
	}
	return &initialAdmin{email: email, passwordHash: hash}, nil
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

// CurrentPrincipal resolves the identity attached to the session loaded into
// ctx by SCS middleware.
func (s *Service) CurrentPrincipal(ctx context.Context) (*Principal, error) {
	kind := s.sessions.GetString(ctx, sessionPrincipalKindKey)
	hasUserID := s.sessions.Exists(ctx, sessionUserIDKey)
	userID := s.sessions.GetInt64(ctx, sessionUserIDKey)

	switch kind {
	case "":
		if hasUserID {
			s.destroyInvalidSession(ctx)
		}
		return nil, ErrNotAuthenticated
	case principalKindInitialAdmin:
		if hasUserID || s.initialAdmin == nil {
			s.destroyInvalidSession(ctx)
			return nil, ErrNotAuthenticated
		}
		return s.initialAdmin.principal(), nil
	case principalKindPersistedUser:
		if !hasUserID || userID == 0 {
			s.destroyInvalidSession(ctx)
			return nil, ErrNotAuthenticated
		}
	default:
		s.destroyInvalidSession(ctx)
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
	return principalFromUser(user), nil
}

// Authenticate resolves the caller from either a Bearer API key in
// authHeader or, when authHeader is empty or malformed, the session cookie
// already loaded into ctx by SCS middleware. Returns ErrNotAuthenticated for
// both missing and bad credentials.
func (s *Service) Authenticate(ctx context.Context, authHeader string) (*Principal, error) {
	if token, ok := httpx.BearerToken(authHeader); ok {
		return s.principalByAPIKey(ctx, token)
	}
	return s.CurrentPrincipal(ctx)
}

// Logout revokes the active session.
func (s *Service) Logout(ctx context.Context) error {
	if err := s.sessions.Destroy(ctx); err != nil {
		return fmt.Errorf("destroy session: %w", err)
	}
	return nil
}

func (s *Service) startPersistedSession(ctx context.Context, userID int64) error {
	return s.startSession(ctx, principalKindPersistedUser, userID)
}

func (s *Service) startInitialAdminSession(ctx context.Context) error {
	return s.startSession(ctx, principalKindInitialAdmin, 0)
}

func (s *Service) startSession(ctx context.Context, kind string, userID int64) error {
	if err := s.sessions.RenewToken(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}
	s.sessions.Remove(ctx, sessionPrincipalKindKey)
	s.sessions.Remove(ctx, sessionUserIDKey)
	s.sessions.Put(ctx, sessionPrincipalKindKey, kind)
	if kind == principalKindPersistedUser {
		s.sessions.Put(ctx, sessionUserIDKey, userID)
	}
	return nil
}

func (s *Service) destroyInvalidSession(ctx context.Context) {
	_ = s.sessions.Destroy(ctx)
}

func (configured *initialAdmin) principal() *Principal {
	return &Principal{
		Email: configured.email,
		Role:  directory.RoleAdmin,
	}
}

func principalFromUser(user *directory.User) *Principal {
	return &Principal{
		UserID: &user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Role:   *user.Role,
	}
}
