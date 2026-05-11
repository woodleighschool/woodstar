package secrets

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/db/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Secret is a reusable shared credential shown to admins.
type Secret struct {
	ID        int64     `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

// Store persists shared credentials.
type Store struct {
	q *sqlc.Queries
}

// NewStore returns a secret store backed by db.
func NewStore(db *db.DB) *Store {
	return &Store{q: db.Queries()}
}

// ListOrbitEnrollSecrets returns active Orbit enrollment tokens ordered newest first.
func (s *Store) ListOrbitEnrollSecrets(ctx context.Context) ([]Secret, error) {
	rows, err := s.q.ListSecrets(ctx, sqlc.ListSecretsParams{Kind: sqlc.SecretKindOrbit})
	if err != nil {
		return nil, err
	}

	secrets := make([]Secret, 0, len(rows))
	for _, row := range rows {
		secrets = append(secrets, secretFromRecord(row))
	}
	return secrets, nil
}

// CreateOrbitEnrollSecret generates and stores a new Orbit enrollment token.
func (s *Store) CreateOrbitEnrollSecret(ctx context.Context) (Secret, error) {
	value, err := randomSecret()
	if err != nil {
		return Secret{}, err
	}

	row, err := s.q.CreateSecret(ctx, sqlc.CreateSecretParams{
		Kind:  sqlc.SecretKindOrbit,
		Value: value,
	})
	if err != nil {
		return Secret{}, err
	}
	return secretFromRecord(row), nil
}

// ValidateOrbitEnrollSecret returns the active Orbit enrollment token matching value.
// Comparison is plaintext because admins can view and reuse these secrets.
func (s *Store) ValidateOrbitEnrollSecret(ctx context.Context, value string) (Secret, bool, error) {
	if strings.TrimSpace(value) == "" {
		return Secret{}, false, nil
	}

	secrets, err := s.ListOrbitEnrollSecrets(ctx)
	if err != nil {
		return Secret{}, false, err
	}
	for _, secret := range secrets {
		if secret.Value == value {
			return secret, true, nil
		}
	}
	return Secret{}, false, nil
}

// HasActiveOrbitEnrollSecret reports whether value matches an active Orbit enrollment token.
func (s *Store) HasActiveOrbitEnrollSecret(ctx context.Context, value string) (bool, error) {
	if strings.TrimSpace(value) == "" {
		return false, nil
	}
	return s.q.HasActiveSecret(ctx, sqlc.HasActiveSecretParams{
		Kind:  sqlc.SecretKindOrbit,
		Value: value,
	})
}

// DeleteOrbitEnrollSecret soft-deletes an Orbit enrollment token by ID.
func (s *Store) DeleteOrbitEnrollSecret(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}

	_, err := s.q.DeleteSecret(ctx, sqlc.DeleteSecretParams{
		ID:   id,
		Kind: sqlc.SecretKindOrbit,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func secretFromRecord(row sqlc.Secret) Secret {
	return Secret{
		ID:        row.ID,
		Value:     row.Value,
		CreatedAt: row.CreatedAt,
	}
}

func randomSecret() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
