package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alexedwards/scs/v2"

	"github.com/woodleighschool/woodstar/internal/models"
)

const sessionUserKey = "user_id"

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
	users    userStore
	sessions *scs.SessionManager
}

type userStore interface {
	Exists(context.Context) (bool, error)
	Create(context.Context, models.CreateUserParams) (*models.User, error)
	GetByEmail(context.Context, string) (*models.User, error)
	GetByID(context.Context, int64) (*models.User, error)
	List(context.Context) ([]models.User, error)
	Update(context.Context, int64, models.UpdateUserParams) (*models.User, error)
	SoftDelete(context.Context, int64) error
	CountAdmins(context.Context) (int, error)
}

// SetupParams contains the first administrator account fields.
type SetupParams struct {
	Email    string
	Name     string
	Password string
}

// NewService creates an auth service backed by a user store and an scs session manager.
func NewService(users userStore, sessions *scs.SessionManager) *Service {
	return &Service{users: users, sessions: sessions}
}

// SetupComplete reports whether the initial administrator account exists.
func (s *Service) SetupComplete(ctx context.Context) (bool, error) {
	if s.users == nil {
		return false, nil
	}
	return s.users.Exists(ctx)
}

// Setup creates the first administrator account and starts a session.
func (s *Service) Setup(ctx context.Context, params SetupParams) (*models.User, error) {
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

	hash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.users.Create(ctx, models.CreateUserParams{
		Email:        params.Email,
		Name:         fallbackName(params.Name, params.Email),
		PasswordHash: hash,
		Role:         models.RoleAdmin,
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
func (s *Service) Login(ctx context.Context, email string, password string) (*models.User, error) {
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

	if err := s.startSession(ctx, user.ID); err != nil {
		return nil, err
	}
	return user, nil
}

// CurrentUser returns the user attached to the session loaded into ctx by scs middleware.
func (s *Service) CurrentUser(ctx context.Context) (*models.User, error) {
	if s.users == nil || s.sessions == nil {
		return nil, ErrNotAuthenticated
	}
	id := s.sessions.GetInt64(ctx, sessionUserKey)
	if id == 0 {
		return nil, ErrNotAuthenticated
	}
	user, err := s.users.GetByID(ctx, id)
	if errors.Is(err, models.ErrNotFound) {
		// Session pointed at a deleted user; clear it.
		_ = s.sessions.Destroy(ctx)
		return nil, ErrNotAuthenticated
	}
	if err != nil {
		return nil, fmt.Errorf("get session user: %w", err)
	}
	return user, nil
}

// Logout revokes the active session.
func (s *Service) Logout(ctx context.Context) error {
	if s.sessions == nil {
		return nil
	}
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

func fallbackName(name string, email string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(email)
}
