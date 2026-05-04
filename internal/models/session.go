package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// SessionStore persists browser admin sessions.
type SessionStore struct {
	q *sqlc.Queries
}

// NewSessionStore returns a session store backed by db.
func NewSessionStore(db *database.DB) *SessionStore {
	return &SessionStore{q: db.Queries()}
}

// Create stores a new browser session.
func (s *SessionStore) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	return s.q.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
}

// UserByTokenHash returns the active user for tokenHash and refreshes last seen time.
func (s *SessionStore) UserByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*User, error) {
	row, err := s.q.GetUserByActiveSessionToken(ctx, sqlc.GetUserByActiveSessionTokenParams{
		TokenHash: tokenHash,
		SeenAt:    &now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return userFromRecord(row), nil
}

// Revoke marks a browser session as revoked.
func (s *SessionStore) Revoke(ctx context.Context, tokenHash string) error {
	return s.q.RevokeSession(ctx, sqlc.RevokeSessionParams{TokenHash: tokenHash})
}

// RevokeAllForUser marks every active session belonging to userID as revoked.
func (s *SessionStore) RevokeAllForUser(ctx context.Context, userID int64) error {
	return s.q.RevokeSessionsForUser(ctx, sqlc.RevokeSessionsForUserParams{UserID: userID})
}
