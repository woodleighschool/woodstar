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
	return new(userFromSQLC(row)), nil
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
	return new(userFromSQLC(row)), nil
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
	return new(userFromSQLC(row)), nil
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
	return new(userFromSQLC(row)), nil
}

// Delete removes the user with id.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteUser(ctx, sqlc.DeleteUserParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

// GetByAPIKey returns the user owning the given API key, or ErrNotFound.
func (s *Store) GetByAPIKey(ctx context.Context, key string) (*User, error) {
	row, err := s.q.GetUserByAPIKey(ctx, sqlc.GetUserByAPIKeyParams{APIKey: &key})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

// SetAPIKey writes a freshly generated API key for id and resets the
// created_at and last_used_at timestamps.
func (s *Store) SetAPIKey(ctx context.Context, id int64, key string) (*User, error) {
	row, err := s.q.SetUserAPIKey(ctx, sqlc.SetUserAPIKeyParams{ID: id, APIKey: &key})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, dbutil.ErrAlreadyExists
		}
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

// ClearAPIKey removes the API key for id.
func (s *Store) ClearAPIKey(ctx context.Context, id int64) (*User, error) {
	row, err := s.q.ClearUserAPIKey(ctx, sqlc.ClearUserAPIKeyParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func userFromSQLC(s sqlc.User) User {
	u := User{
		ID:              s.ID,
		Email:           s.Email,
		Name:            s.Name,
		PasswordHash:    s.PasswordHash,
		Role:            s.Role,
		APIKeyCreatedAt: s.APIKeyCreatedAt,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
	if s.APIKey != nil {
		u.APIKey = *s.APIKey
	}
	return u
}
