package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

// apiKeyByteLen provides 192 bits of entropy and encodes to 32 base64url characters.
const apiKeyByteLen = 24

// RotateAPIKey replaces userID's retrievable API key and returns the updated account.
func (s *Service) RotateAPIKey(ctx context.Context, userID int64) (*directory.Account, error) {
	key, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		return nil, err
	}
	account, err := s.users.SetAccountAPIKey(ctx, userID, key)
	if err != nil {
		return nil, fmt.Errorf("set API key: %w", err)
	}
	return account, nil
}

// RevokeAPIKey clears the API key on userID and returns the updated account.
func (s *Service) RevokeAPIKey(ctx context.Context, userID int64) (*directory.Account, error) {
	account, err := s.users.ClearAccountAPIKey(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("clear API key: %w", err)
	}
	return account, nil
}

func (s *Service) userByAPIKey(ctx context.Context, token string) (*directory.User, error) {
	if token == "" {
		return nil, ErrNotAuthenticated
	}
	user, err := s.users.GetByAPIKey(ctx, token)
	if errors.Is(err, fault.ErrNotFound) {
		return nil, ErrNotAuthenticated
	}
	if err != nil {
		return nil, fmt.Errorf("get user by API key: %w", err)
	}
	return user, nil
}
