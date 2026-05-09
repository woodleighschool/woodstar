package models

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// UserRole controls application permissions.
type UserRole = sqlc.UserRole

// User roles are intentionally small.
const (
	RoleAdmin  = sqlc.UserRoleAdmin
	RoleViewer = sqlc.UserRoleViewer
)

// User is a local account.
type User struct {
	sqlc.User
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
		Role:         params.Role,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return &User{User: row}, nil
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
	return &User{User: row}, nil
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
	return &User{User: row}, nil
}

// List returns active users ordered by creation time.
func (s *UserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = User{User: row}
	}
	return users, nil
}

// Update writes the writable fields of a user. Name and Role are required;
// PasswordHash is optional and left untouched when nil.
func (s *UserStore) Update(ctx context.Context, id int64, params UpdateUserParams) (*User, error) {
	row, err := s.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		Name:         strings.TrimSpace(params.Name),
		Role:         params.Role,
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
	return &User{User: row}, nil
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
