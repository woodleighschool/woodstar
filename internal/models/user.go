package models

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
)

// UserRole controls a Woodstar user's application permissions.
type UserRole string

// User roles are intentionally small for MVP.
const (
	RoleAdmin  UserRole = "admin"
	RoleViewer UserRole = "viewer"
)

// User is a local Woodstar account.
type User struct {
	ID           int64
	Email        string
	Name         string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserStore persists local Woodstar accounts.
type UserStore struct {
	db *database.DB
}

// CreateUserParams contains fields needed to create a local account.
type CreateUserParams struct {
	Email        string
	Name         string
	PasswordHash string
	Role         UserRole
}

// NewUserStore returns a user store backed by db.
func NewUserStore(db *database.DB) *UserStore {
	return &UserStore{db: db}
}

// Exists reports whether any active local user exists.
func (s *UserStore) Exists(ctx context.Context) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users WHERE deleted_at IS NULL)").Scan(&exists)
	return exists, err
}

// Create inserts a local user.
func (s *UserStore) Create(ctx context.Context, params CreateUserParams) (*User, error) {
	user := &User{}
	var role string
	err := s.db.QueryRow(ctx, `
INSERT INTO users (email, name, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING id, email, name, password_hash, role, created_at, updated_at`,
		normalizeEmail(params.Email),
		strings.TrimSpace(params.Name),
		params.PasswordHash,
		string(params.Role),
	).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	user.Role = UserRole(role)
	return user, nil
}

// GetByEmail returns an active user by email.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.get(ctx, "email = $1", normalizeEmail(email))
}

// GetByID returns an active user by database ID.
func (s *UserStore) GetByID(ctx context.Context, id int64) (*User, error) {
	return s.get(ctx, "id = $1", id)
}

func (s *UserStore) get(ctx context.Context, where string, arg any) (*User, error) {
	user := &User{}
	var role string
	err := s.db.QueryRow(ctx, `
SELECT id, email, name, password_hash, role, created_at, updated_at
FROM users
WHERE `+where+` AND deleted_at IS NULL`,
		arg,
	).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &role, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	user.Role = UserRole(role)
	return user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// UserIDString formats a database user ID for API responses.
func UserIDString(id int64) string {
	return strconv.FormatInt(id, 10)
}
