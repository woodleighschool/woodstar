package auth

import (
	"context"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/directory"
)

// Account returns the signed-in user's self-view, including their retrievable API key.
func (s *Service) Account(ctx context.Context, userID int64) (*directory.Account, error) {
	account, err := s.users.GetAccount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return account, nil
}
