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

// Store persists local accounts.
type Store struct {
	q *sqlc.Queries
}

// UserCreate contains fields needed to create a user.
type UserCreate struct {
	Email    string `json:"email"          format:"email"`
	Name     string `json:"name,omitempty"`
	Role     Role   `json:"role"`
	Password string `json:"password"                      minLength:"12"`
}

// UserMutation replaces the writable fields of a user.
type UserMutation struct {
	Name     string  `json:"name"`
	Role     Role    `json:"role"`
	Password *string `json:"password,omitempty"`
}

func NewStore(db *database.DB) *Store {
	return &Store{q: db.Queries()}
}

func (s *Store) Exists(ctx context.Context) (bool, error) {
	return s.q.UserExists(ctx)
}

func (s *Store) Create(ctx context.Context, params UserCreate) (*User, error) {
	hash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	row, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        strings.ToLower(params.Email),
		Name:         params.Name,
		PasswordHash: hash,
		Role:         sqlc.UserRole(params.Role),
	})
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, dbutil.ErrAlreadyExists
		}
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) GetByEmail(ctx context.Context, email string) (*User, error) {
	row, err := s.q.GetUserByEmail(ctx, sqlc.GetUserByEmailParams{Email: strings.ToLower(email)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

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

// GetAccountByID returns the signed-in user's self-view, including API key fields.
func (s *Store) GetAccountByID(ctx context.Context, id int64) (*Account, error) {
	row, err := s.q.GetUserByID(ctx, sqlc.GetUserByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(accountFromSQLC(row)), nil
}

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

func (s *Store) Update(ctx context.Context, id int64, params UserMutation) (*User, error) {
	var passwordHash *string
	if params.Password != nil {
		hash, err := HashPassword(*params.Password)
		if err != nil {
			return nil, err
		}
		passwordHash = &hash
	}

	row, err := s.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		Name:         params.Name,
		Role:         sqlc.UserRole(params.Role),
		PasswordHash: passwordHash,
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

// UpdateAccount writes the signed-in user's editable account fields and
// returns the fresh account view in a single round trip.
func (s *Store) UpdateAccount(ctx context.Context, id int64, params AccountMutation) (*Account, error) {
	var passwordHash *string
	if params.Password != nil {
		hash, err := HashPassword(*params.Password)
		if err != nil {
			return nil, err
		}
		passwordHash = &hash
	}
	row, err := s.q.UpdateAccountByID(ctx, sqlc.UpdateAccountByIDParams{
		Name:         params.Name,
		PasswordHash: passwordHash,
		ID:           id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(accountFromSQLC(row)), nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteUser(ctx, sqlc.DeleteUserParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

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
func (s *Store) SetAPIKey(ctx context.Context, id int64, key string) (*Account, error) {
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
	return new(accountFromSQLC(row)), nil
}

func (s *Store) ClearAPIKey(ctx context.Context, id int64) (*Account, error) {
	row, err := s.q.ClearUserAPIKey(ctx, sqlc.ClearUserAPIKeyParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(accountFromSQLC(row)), nil
}

func userFromSQLC(s sqlc.User) User {
	return User{
		ID:           s.ID,
		Email:        s.Email,
		Name:         s.Name,
		PasswordHash: s.PasswordHash,
		Role:         Role(s.Role),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func accountFromSQLC(s sqlc.User) Account {
	account := Account{
		User:            userFromSQLC(s),
		APIKeyCreatedAt: s.APIKeyCreatedAt,
	}
	if s.APIKey != nil {
		account.APIKey = *s.APIKey
	}
	return account
}
