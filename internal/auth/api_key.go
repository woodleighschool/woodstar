package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/randtoken"
	"github.com/woodleighschool/woodstar/internal/users"
)

// apiKeyByteLen is the entropy budget for the random component of every
// generated key. 24 bytes encode to 32 url-safe base64 characters.
const apiKeyByteLen = 24

// RotateAPIKey generates a fresh API key for userID, persists it, and returns
// the updated account self-view with the retrievable plaintext key.
func (s *Service) RotateAPIKey(ctx context.Context, userID int64) (*users.Account, error) {
	key, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		return nil, err
	}
	account, err := s.users.SetAPIKey(ctx, userID, key)
	if err != nil {
		return nil, fmt.Errorf("set api key: %w", err)
	}
	return account, nil
}

// RevokeAPIKey clears the API key on userID and returns the updated user.
func (s *Service) RevokeAPIKey(ctx context.Context, userID int64) (*users.Account, error) {
	account, err := s.users.ClearAPIKey(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("clear api key: %w", err)
	}
	return account, nil
}

// userByAPIKey returns the user owning the given API key and best-effort
// updates api_key_last_used_at. Token lookup is plain equality on the
// indexed column; the user is not retrieved when token is empty.
func (s *Service) userByAPIKey(ctx context.Context, token string) (*users.User, error) {
	if token == "" {
		return nil, ErrNotAuthenticated
	}
	user, err := s.users.GetByAPIKey(ctx, token)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, ErrNotAuthenticated
	}
	if err != nil {
		return nil, fmt.Errorf("get user by api key: %w", err)
	}
	return user, nil
}
