package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/models"
)

// SessionCookieName is the browser cookie used for Woodstar admin sessions.
const SessionCookieName = "woodstar_session"

// Auth errors describe expected setup and login failures.
var (
	ErrAlreadySetup       = errors.New("woodstar is already set up")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotAuthenticated   = errors.New("not authenticated")
	ErrNotSetup           = errors.New("woodstar setup is required")
	ErrWeakPassword       = errors.New("password must be at least 12 characters")
)

// Service owns local setup, login, and session lookup behavior.
type Service struct {
	users         *models.UserStore
	sessions      *models.SessionStore
	sessionTTL    time.Duration
	sessionSecret string
}

// SetupParams contains the first administrator account fields.
type SetupParams struct {
	Email    string
	Name     string
	Password string
}

// LoginResult contains the authenticated user and browser session token.
type LoginResult struct {
	User      *models.User
	Token     string
	ExpiresAt time.Time
}

// NewService creates an auth service backed by user and session stores.
func NewService(
	users *models.UserStore,
	sessions *models.SessionStore,
	sessionTTL time.Duration,
	sessionSecret string,
) *Service {
	if sessionTTL <= 0 {
		sessionTTL = 14 * 24 * time.Hour
	}
	return &Service{
		users:         users,
		sessions:      sessions,
		sessionTTL:    sessionTTL,
		sessionSecret: sessionSecret,
	}
}

// SetupComplete reports whether the initial administrator account exists.
func (s *Service) SetupComplete(ctx context.Context) (bool, error) {
	if s.users == nil {
		return false, nil
	}
	return s.users.Exists(ctx)
}

// Setup creates the first administrator account and returns a session.
func (s *Service) Setup(ctx context.Context, params SetupParams) (*LoginResult, error) {
	if s.users == nil || s.sessions == nil {
		return nil, ErrNotSetup
	}
	exists, err := s.users.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("check setup state: %w", err)
	}
	if exists {
		return nil, ErrAlreadySetup
	}

	passwordHash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.users.Create(ctx, models.CreateUserParams{
		Email:        params.Email,
		Name:         fallbackName(params.Name, params.Email),
		PasswordHash: passwordHash,
		Role:         models.RoleAdmin,
	})
	if err != nil {
		return nil, fmt.Errorf("create setup user: %w", err)
	}

	return s.createSession(ctx, user)
}

// Login checks local credentials and returns a session.
func (s *Service) Login(ctx context.Context, email string, password string) (*LoginResult, error) {
	if s.users == nil || s.sessions == nil {
		return nil, ErrNotSetup
	}
	exists, err := s.users.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("check setup state: %w", err)
	}
	if !exists {
		return nil, ErrNotSetup
	}

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, models.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	valid, err := VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return nil, ErrInvalidCredentials
	}

	return s.createSession(ctx, user)
}

// CurrentUser returns the user attached to an active session token.
func (s *Service) CurrentUser(ctx context.Context, token string) (*models.User, error) {
	token = strings.TrimSpace(token)
	if token == "" || s.sessions == nil {
		return nil, ErrNotAuthenticated
	}

	user, err := s.sessions.UserByTokenHash(ctx, s.tokenHash(token), time.Now().UTC())
	if errors.Is(err, models.ErrNotFound) {
		return nil, ErrNotAuthenticated
	}
	if err != nil {
		return nil, fmt.Errorf("get session user: %w", err)
	}
	return user, nil
}

// Logout revokes a session token.
func (s *Service) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" || s.sessions == nil {
		return nil
	}
	if err := s.sessions.Revoke(ctx, s.tokenHash(token)); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Service) createSession(ctx context.Context, user *models.User) (*LoginResult, error) {
	token, err := sessionToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	if err := s.sessions.Create(ctx, user.ID, s.tokenHash(token), expiresAt); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &LoginResult{
		User:      user,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) tokenHash(token string) string {
	return hashToken(s.sessionSecret, token)
}

func sessionToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func fallbackName(name string, email string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(email)
}
