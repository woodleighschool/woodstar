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

// UpdateUserParams contains the optional fields an admin can change on a user.
// Nil fields are left untouched.
type UpdateUserParams struct {
	Name         *string
	Role         *UserRole
	PasswordHash *string
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

// List returns active users ordered by creation time.
func (s *UserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.db.Query(ctx, `
SELECT id, email, name, password_hash, role, created_at, updated_at
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		var role string
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.PasswordHash,
			&role,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		user.Role = UserRole(role)
		users = append(users, user)
	}
	return users, rows.Err()
}

// Update applies the non-nil fields of params to the user with id and returns the row.
func (s *UserStore) Update(ctx context.Context, id int64, params UpdateUserParams) (*User, error) {
	var nameArg any
	if params.Name != nil {
		nameArg = strings.TrimSpace(*params.Name)
	}
	var roleArg any
	if params.Role != nil {
		roleArg = string(*params.Role)
	}
	var passwordArg any
	if params.PasswordHash != nil {
		passwordArg = *params.PasswordHash
	}

	user := &User{}
	var role string
	err := s.db.QueryRow(ctx, `
UPDATE users
SET name          = COALESCE($2, name),
    role          = COALESCE($3::user_role, role),
    password_hash = COALESCE($4, password_hash),
    updated_at    = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, email, name, password_hash, role, created_at, updated_at`,
		id, nameArg, roleArg, passwordArg,
	).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &role, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	user.Role = UserRole(role)
	return user, nil
}

// SoftDelete marks the user with id as deleted.
func (s *UserStore) SoftDelete(ctx context.Context, id int64) error {
	var deletedID int64
	err := s.db.QueryRow(ctx, `
UPDATE users
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id`, id).Scan(&deletedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// CountAdmins returns the number of active admin users.
func (s *UserStore) CountAdmins(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE role = 'admin' AND deleted_at IS NULL`,
	).Scan(&count)
	return count, err
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
