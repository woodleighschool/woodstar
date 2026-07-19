package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

// apiKeyByteLen is the entropy budget for the random component of every
// generated key. 24 bytes encode to 32 url-safe base64 characters.
const apiKeyByteLen = 24

// RotateAPIKey generates a fresh API key for userID, persists it, and returns
// the updated account self-view with the retrievable plaintext key.
func (s *Service) RotateAPIKey(ctx context.Context, userID int64) (*directory.Account, error) {
	key, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		return nil, err
	}
	account, err := s.users.SetAccountAPIKey(ctx, userID, key)
	if err != nil {
		return nil, fmt.Errorf("set api key: %w", err)
	}
	return account, nil
}

// RevokeAPIKey clears the API key on userID and returns the updated account.
func (s *Service) RevokeAPIKey(ctx context.Context, userID int64) (*directory.Account, error) {
	account, err := s.users.ClearAccountAPIKey(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("clear api key: %w", err)
	}
	return account, nil
}

// principalByAPIKey returns the persisted principal owning the given API key.
// Token lookup is plain equality on the indexed column; the user is not
// retrieved when token is empty.
func (s *Service) principalByAPIKey(ctx context.Context, token string) (*Principal, error) {
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
	return principalFromUser(user), nil
}
