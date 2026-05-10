package models

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/db/sqlc"
	"github.com/woodleighschool/woodstar/internal/store"
)

// SecretKind identifies the subsystem that accepts a shared secret.
type SecretKind string

// Secret kinds map to agent-facing credentials.
// Santa and Munki return alongside their modules.
const (
	SecretOrbit SecretKind = "orbit"
)

// Secret is a reusable shared credential shown to admins.
type Secret struct {
	ID        int64     `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

// SecretStore persists reusable shared credentials.
type SecretStore struct {
	q *sqlc.Queries
}

// NewSecretStore returns a secret store backed by db.
func NewSecretStore(db *db.DB) *SecretStore {
	return &SecretStore{q: db.Queries()}
}

// List returns active secrets of kind ordered newest first.
func (s *SecretStore) List(ctx context.Context, kind SecretKind) ([]Secret, error) {
	if err := kind.Valid(); err != nil {
		return nil, err
	}

	rows, err := s.q.ListSecrets(ctx, sqlc.ListSecretsParams{Kind: sqlc.SecretKind(kind)})
	if err != nil {
		return nil, err
	}

	secrets := make([]Secret, 0, len(rows))
	for _, row := range rows {
		secrets = append(secrets, secretFromRecord(row))
	}
	return secrets, nil
}

// Create generates and stores a new secret of kind.
func (s *SecretStore) Create(ctx context.Context, kind SecretKind) (*Secret, error) {
	if err := kind.Valid(); err != nil {
		return nil, err
	}

	value, err := randomSecret()
	if err != nil {
		return nil, err
	}

	row, err := s.q.CreateSecret(ctx, sqlc.CreateSecretParams{
		Kind:  sqlc.SecretKind(kind),
		Value: value,
	})
	if err != nil {
		return nil, err
	}
	secret := secretFromRecord(row)
	return &secret, nil
}

// ValidateActive reports whether value matches an active secret of kind.
// Comparison is plaintext because admins can view and reuse these secrets.
func (s *SecretStore) ValidateActive(ctx context.Context, kind SecretKind, value string) (bool, error) {
	if err := kind.Valid(); err != nil {
		return false, err
	}
	if strings.TrimSpace(value) == "" {
		return false, nil
	}

	return s.q.HasActiveSecret(ctx, sqlc.HasActiveSecretParams{
		Kind:  sqlc.SecretKind(kind),
		Value: value,
	})
}

// Delete soft-deletes a secret by kind and ID.
func (s *SecretStore) Delete(ctx context.Context, kind SecretKind, id int64) error {
	if err := kind.Valid(); err != nil {
		return err
	}
	if id <= 0 {
		return store.ErrNotFound
	}

	_, err := s.q.DeleteSecret(ctx, sqlc.DeleteSecretParams{
		ID:   id,
		Kind: sqlc.SecretKind(kind),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ErrNotFound
	}
	return err
}

// Valid reports whether k is a supported secret kind.
func (k SecretKind) Valid() error {
	switch k {
	case SecretOrbit:
		return nil
	default:
		return fmt.Errorf("unknown secret kind %q", k)
	}
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
