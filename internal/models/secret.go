package models

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
)

// SecretKind identifies the subsystem that accepts a shared secret.
type SecretKind string

// Secret kinds map to agent-facing credentials.
const (
	SecretOrbit SecretKind = "orbit"
	SecretSanta SecretKind = "santa"
	SecretMunki SecretKind = "munki"
)

// Secret is a reusable shared credential shown to admins.
type Secret struct {
	ID        string    `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

// SecretStore persists reusable shared credentials.
type SecretStore struct {
	db *database.DB
}

// NewSecretStore returns a secret store backed by db.
func NewSecretStore(db *database.DB) *SecretStore {
	return &SecretStore{db: db}
}

// List returns active secrets of kind ordered newest first.
func (s *SecretStore) List(ctx context.Context, kind SecretKind) ([]Secret, error) {
	if err := kind.Valid(); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
SELECT id, value, created_at
FROM secrets
WHERE kind = $1 AND deleted_at IS NULL
ORDER BY created_at DESC`, string(kind))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	secrets := make([]Secret, 0)
	for rows.Next() {
		var secret Secret
		var id int64
		if err := rows.Scan(&id, &secret.Value, &secret.CreatedAt); err != nil {
			return nil, err
		}
		secret.ID = strconv.FormatInt(id, 10)
		secrets = append(secrets, secret)
	}

	return secrets, rows.Err()
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

	secret := &Secret{Value: value}
	var id int64
	err = s.db.QueryRow(ctx, `
INSERT INTO secrets (kind, value)
VALUES ($1, $2)
RETURNING id, created_at`, string(kind), value).Scan(&id, &secret.CreatedAt)
	if err != nil {
		return nil, err
	}

	secret.ID = strconv.FormatInt(id, 10)
	return secret, nil
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

	var exists bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM secrets WHERE kind = $1 AND value = $2 AND deleted_at IS NULL)`,
		string(kind), value,
	).Scan(&exists)
	return exists, err
}

// Delete soft-deletes a secret by kind and API ID.
func (s *SecretStore) Delete(ctx context.Context, kind SecretKind, id string) error {
	if err := kind.Valid(); err != nil {
		return err
	}

	parsedID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsedID <= 0 {
		return ErrNotFound
	}

	err = s.db.QueryRow(ctx, `
UPDATE secrets
SET deleted_at = now()
WHERE id = $1 AND kind = $2 AND deleted_at IS NULL
RETURNING id`, parsedID, string(kind)).Scan(&parsedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// Valid reports whether k is a supported secret kind.
func (k SecretKind) Valid() error {
	switch k {
	case SecretOrbit, SecretSanta, SecretMunki:
		return nil
	default:
		return fmt.Errorf("unknown secret kind %q", k)
	}
}

func randomSecret() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
