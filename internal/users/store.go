package users

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// User roles are intentionally small.
const (
	RoleAdmin  = sqlc.UserRoleAdmin
	RoleViewer = sqlc.UserRoleViewer
)

// Store persists local accounts.
type Store struct {
	q *sqlc.Queries
}

// CreateRecordParams contains fields needed to persist a local account.
type CreateRecordParams struct {
	Email        string
	Name         string
	PasswordHash string
	Role         Role
}

// UpdateRecordParams replaces the writable fields of a user.
type UpdateRecordParams struct {
	Name         string
	Role         Role
	PasswordHash *string
}

// NewStore returns a user store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{q: db.Queries()}
}

// Exists reports whether any active local user exists.
func (s *Store) Exists(ctx context.Context) (bool, error) {
	return s.q.UserExists(ctx)
}

// Create inserts a local user.
func (s *Store) Create(ctx context.Context, params CreateRecordParams) (*User, error) {
	row, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        normalizeEmail(params.Email),
		Name:         strings.TrimSpace(params.Name),
		PasswordHash: params.PasswordHash,
		Role:         params.Role,
	})
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, dbutil.ErrAlreadyExists
		}
		return nil, err
	}
	u := userFromSQLC(row)
	return &u, nil
}

// GetByEmail returns an active user by email.
func (s *Store) GetByEmail(ctx context.Context, email string) (*User, error) {
	row, err := s.q.GetUserByEmail(ctx, sqlc.GetUserByEmailParams{Email: normalizeEmail(email)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u := userFromSQLC(row)
	return &u, nil
}

// GetByID returns an active user by database ID.
func (s *Store) GetByID(ctx context.Context, id int64) (*User, error) {
	row, err := s.q.GetUserByID(ctx, sqlc.GetUserByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u := userFromSQLC(row)
	return &u, nil
}

// List returns active users ordered by creation time.
func (s *Store) List(ctx context.Context) ([]User, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = userFromSQLC(row)
	}
	return users, nil
}

// Update writes the writable fields of a user. Name and Role are required;
// PasswordHash is optional and left untouched when nil.
func (s *Store) Update(ctx context.Context, id int64, params UpdateRecordParams) (*User, error) {
	row, err := s.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		Name:         strings.TrimSpace(params.Name),
		Role:         params.Role,
		PasswordHash: params.PasswordHash,
		ID:           id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, dbutil.ErrAlreadyExists
		}
		return nil, err
	}
	u := userFromSQLC(row)
	return &u, nil
}

// Delete removes the user with id.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteUser(ctx, sqlc.DeleteUserParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

// CountAdmins returns the number of active admin users.
func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	count, err := s.q.CountAdminUsers(ctx)
	return int(count), err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func userFromSQLC(s sqlc.User) User {
	return User{
		ID:           s.ID,
		Email:        s.Email,
		Name:         s.Name,
		PasswordHash: s.PasswordHash,
		Role:         s.Role,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}
