package directory

import (
	"context"
	"errors"

	"github.com/woodleighschool/goodies/auth/authn"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

// AuthnStore reads authentication identities without loading directory profiles or roles.
type AuthnStore struct {
	store *Store
}

// NewAuthnStore returns an authentication store backed by store.
func NewAuthnStore(store *Store) *AuthnStore {
	return &AuthnStore{store: store}
}

func (s *AuthnStore) GetPrincipal(ctx context.Context, id int64) (*authn.Principal, error) {
	principal, err := postgres.GetOne[authn.Principal](ctx, s.store.pool, `
SELECT id, email, name FROM users
WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, authnStoreError(err)
	}
	return &principal, nil
}

func (s *AuthnStore) GetPrincipalByAPIKey(ctx context.Context, key string) (*authn.Principal, error) {
	principal, err := postgres.GetOne[authn.Principal](ctx, s.store.pool, `
SELECT id, email, name FROM users
WHERE api_key = $1 AND deleted_at IS NULL`, key)
	if err != nil {
		return nil, authnStoreError(err)
	}
	return &principal, nil
}

func (s *AuthnStore) GetPasswordIdentityByEmail(
	ctx context.Context,
	email string,
) (*authn.PasswordIdentity, error) {
	identity, err := postgres.GetOne[authn.PasswordIdentity](ctx, s.store.pool, `
SELECT id, email, name, password_hash FROM users
WHERE email = $1 AND source = 'local'
  AND password_hash IS NOT NULL AND deleted_at IS NULL`, email)
	if err != nil {
		return nil, authnStoreError(err)
	}
	return &identity, nil
}

func (s *AuthnStore) GetSSOPrincipalByEmail(ctx context.Context, email string) (*authn.Principal, error) {
	principal, err := postgres.GetOne[authn.Principal](ctx, s.store.pool, `
SELECT id, email, name FROM users
WHERE email = $1 AND source <> 'local' AND deleted_at IS NULL`, email)
	if err != nil {
		return nil, authnStoreError(err)
	}
	return &principal, nil
}

func (s *AuthnStore) SetAPIKey(ctx context.Context, id int64, key string) error {
	tag, err := s.store.pool.Exec(ctx, `
UPDATE users SET api_key = $2, api_key_created_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL`, id, key)
	if err != nil {
		return authnStoreError(postgres.MutationError(err))
	}
	if tag.RowsAffected() == 0 {
		return authn.ErrPrincipalNotFound
	}
	return nil
}

func (s *AuthnStore) ClearAPIKey(ctx context.Context, id int64) error {
	tag, err := s.store.pool.Exec(ctx, `
UPDATE users SET api_key = NULL, api_key_created_at = NULL, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return authnStoreError(postgres.MutationError(err))
	}
	if tag.RowsAffected() == 0 {
		return authn.ErrPrincipalNotFound
	}
	return nil
}

func authnStoreError(err error) error {
	if errors.Is(err, fault.ErrNotFound) {
		return authn.ErrPrincipalNotFound
	}
	return err
}
