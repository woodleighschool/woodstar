package models

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// UserRole controls application permissions.
type UserRole string

// User roles are intentionally small.
const (
	RoleAdmin  UserRole = "admin"
	RoleViewer UserRole = "viewer"
)

// User is a local account.
type User struct {
	ID           int64
	Email        string
	Name         string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserStore persists local accounts.
type UserStore struct {
	q *sqlc.Queries
}

// CreateUserParams contains fields needed to create a local account.
type CreateUserParams struct {
	Email        string
	Name         string
	PasswordHash string
	Role         UserRole
}

// UpdateUserParams replaces the writable fields of a user.
type UpdateUserParams struct {
	Name         string
	Role         UserRole
	PasswordHash *string
}

// NewUserStore returns a user store backed by db.
func NewUserStore(db *database.DB) *UserStore {
	return &UserStore{q: db.Queries()}
}

// Exists reports whether any active local user exists.
func (s *UserStore) Exists(ctx context.Context) (bool, error) {
	return s.q.UserExists(ctx)
}

// Create inserts a local user.
func (s *UserStore) Create(ctx context.Context, params CreateUserParams) (*User, error) {
	row, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        normalizeEmail(params.Email),
		Name:         strings.TrimSpace(params.Name),
		PasswordHash: params.PasswordHash,
		Role:         sqlc.UserRole(params.Role),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return userFromRecord(row), nil
}

// GetByEmail returns an active user by email.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	row, err := s.q.GetUserByEmail(ctx, sqlc.GetUserByEmailParams{Email: normalizeEmail(email)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return userFromRecord(row), nil
}

// GetByID returns an active user by database ID.
func (s *UserStore) GetByID(ctx context.Context, id int64) (*User, error) {
	row, err := s.q.GetUserByID(ctx, sqlc.GetUserByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return userFromRecord(row), nil
}

// List returns active users ordered by creation time.
func (s *UserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(rows))
	for _, row := range rows {
		users = append(users, *userFromRecord(row))
	}
	return users, nil
}

// Update writes the writable fields of a user and returns the row.
// Name and Role are required; PasswordHash is optional.
func (s *UserStore) Update(ctx context.Context, id int64, params UpdateUserParams) (*User, error) {
	row, err := s.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		Name:         strings.TrimSpace(params.Name),
		Role:         sqlc.UserRole(params.Role),
		PasswordHash: params.PasswordHash,
		ID:           id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return userFromRecord(row), nil
}

// SoftDelete marks the user with id as deleted.
func (s *UserStore) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.q.SoftDeleteUser(ctx, sqlc.SoftDeleteUserParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// CountAdmins returns the number of active admin users.
func (s *UserStore) CountAdmins(ctx context.Context) (int, error) {
	count, err := s.q.CountAdminUsers(ctx)
	return int(count), err
}

func userFromRecord(row sqlc.User) *User {
	return &User{
		ID:           row.ID,
		Email:        row.Email,
		Name:         row.Name,
		PasswordHash: row.PasswordHash,
		Role:         UserRole(row.Role),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// UserIDString formats a database user ID for API responses.
func UserIDString(id int64) string {
	return strconv.FormatInt(id, 10)
}
