package enrollment

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

// Store persists enrollment credentials accepted by Orbit and osquery.
type Store struct {
	q *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{q: db.Queries()}
}

func (s *Store) List(ctx context.Context) ([]EnrollSecret, error) {
	rows, err := s.q.ListEnrollSecrets(ctx)
	if err != nil {
		return nil, err
	}

	secrets := make([]EnrollSecret, 0, len(rows))
	for _, row := range rows {
		secrets = append(secrets, enrollSecretFromRecord(row))
	}
	return secrets, nil
}

func (s *Store) Create(ctx context.Context) (EnrollSecret, error) {
	value, err := secret.Generate()
	if err != nil {
		return EnrollSecret{}, err
	}

	row, err := s.q.CreateEnrollSecret(ctx, sqlc.CreateEnrollSecretParams{Value: value})
	if err != nil {
		return EnrollSecret{}, err
	}
	return enrollSecretFromRecord(row), nil
}

func (s *Store) HasActive(ctx context.Context, value string) (bool, error) {
	if strings.TrimSpace(value) == "" {
		return false, nil
	}
	return s.q.HasActiveEnrollSecret(ctx, sqlc.HasActiveEnrollSecretParams{Value: value})
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}

	_, err := s.q.DeleteEnrollSecret(ctx, sqlc.DeleteEnrollSecretParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func enrollSecretFromRecord(row sqlc.EnrollSecret) EnrollSecret {
	return EnrollSecret{
		ID:        row.ID,
		Value:     row.Value,
		CreatedAt: row.CreatedAt,
	}
}
