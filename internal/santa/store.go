package santa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists Santa state.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListSyncTokens(ctx context.Context) ([]SyncToken, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id, value_hash, created_at, last_used_at
		FROM santa_sync_tokens
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []SyncToken{}
	for rows.Next() {
		var token SyncToken
		if err := rows.Scan(&token.ID, &token.ValueHash, &token.CreatedAt, &token.LastUsedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s *Store) CreateSyncToken(ctx context.Context) (CreatedSyncToken, error) {
	value, err := randomToken()
	if err != nil {
		return CreatedSyncToken{}, err
	}

	token := CreatedSyncToken{Value: value}
	err = s.db.Pool().QueryRow(ctx, `
		INSERT INTO santa_sync_tokens (value_hash)
		VALUES ($1)
		RETURNING id, value_hash, created_at, last_used_at
	`, hashToken(value)).Scan(
		&token.ID,
		&token.ValueHash,
		&token.CreatedAt,
		&token.LastUsedAt,
	)
	if err != nil {
		return CreatedSyncToken{}, err
	}
	return token, nil
}

func (s *Store) DeleteSyncToken(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}

	var deletedID int64
	err := s.db.Pool().QueryRow(ctx, `
		DELETE FROM santa_sync_tokens
		WHERE id = $1
		RETURNING id
	`, id).Scan(&deletedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) VerifyBearerToken(ctx context.Context, authorization string) (bool, error) {
	value, ok := parseBearerToken(authorization)
	if !ok {
		return false, nil
	}

	valueHash := hashToken(value)
	var exists bool
	if err := s.db.Pool().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM santa_sync_tokens
			WHERE value_hash = $1
		)
	`, valueHash).Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	_, _ = s.db.Pool().Exec(ctx, `
		UPDATE santa_sync_tokens
		SET last_used_at = now()
		WHERE value_hash = $1
	`, valueHash)
	return true, nil
}

func parseBearerToken(authorization string) (string, bool) {
	scheme, value, ok := strings.Cut(authorization, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != "" && !strings.Contains(value, " ")
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
