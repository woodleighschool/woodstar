package syncstate

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/secret"
)

// Store persists Santa sync state and credentials.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) ListTokens(ctx context.Context) ([]SyncToken, error) {
	rows, err := s.q.ListSantaSyncTokens(ctx)
	if err != nil {
		return nil, err
	}
	tokens := make([]SyncToken, len(rows))
	for i, row := range rows {
		tokens[i] = syncTokenFromSQLC(row)
	}
	return tokens, nil
}

func (s *Store) CreateToken(ctx context.Context) (SyncToken, error) {
	value, err := secret.Generate()
	if err != nil {
		return SyncToken{}, err
	}

	token, err := s.q.CreateSantaSyncToken(ctx, sqlc.CreateSantaSyncTokenParams{Value: value})
	if err != nil {
		return SyncToken{}, err
	}
	return syncTokenFromSQLC(token), nil
}

func (s *Store) DeleteToken(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}

	_, err := s.q.DeleteSantaSyncToken(ctx, sqlc.DeleteSantaSyncTokenParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) VerifySyncToken(ctx context.Context, value string) (bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, nil
	}

	return s.q.HasSantaSyncToken(ctx, sqlc.HasSantaSyncTokenParams{Value: value})
}

func syncTokenFromSQLC(token sqlc.SantaSyncToken) SyncToken {
	return SyncToken{
		ID:        token.ID,
		Value:     token.Value,
		CreatedAt: token.CreatedAt,
	}
}
