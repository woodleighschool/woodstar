package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
)

// SessionStore persists browser admin sessions.
type SessionStore struct {
	db *database.DB
}

// NewSessionStore returns a session store backed by db.
func NewSessionStore(db *database.DB) *SessionStore {
	return &SessionStore{db: db}
}

// Create stores a new browser session.
func (s *SessionStore) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	return s.db.Exec(ctx, `
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)`,
		userID,
		tokenHash,
		expiresAt,
	)
}

// UserByTokenHash returns the active user for tokenHash and refreshes last seen time.
func (s *SessionStore) UserByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*User, error) {
	user := &User{}
	var role string
	err := s.db.QueryRow(ctx, `
UPDATE sessions
SET last_seen_at = $2
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > $2
RETURNING user_id`, tokenHash, now).Scan(&user.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow(ctx, `
SELECT id, email, name, password_hash, role, created_at, updated_at
FROM users
WHERE id = $1 AND deleted_at IS NULL`, user.ID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	user.Role = UserRole(role)
	return user, nil
}

// Revoke marks a browser session as revoked.
func (s *SessionStore) Revoke(ctx context.Context, tokenHash string) error {
	return s.db.Exec(ctx, `
UPDATE sessions
SET revoked_at = now()
WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
}

// RevokeAllForUser marks every active session belonging to userID as revoked.
func (s *SessionStore) RevokeAllForUser(ctx context.Context, userID int64) error {
	return s.db.Exec(ctx, `
UPDATE sessions
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL`, userID)
}
