package users

import (
	"context"
	"fmt"
)

// AccountUpdateParams contains fields a signed-in user can mutate on their own account.
type AccountUpdateParams struct {
	Name     string
	Password *string
}

// GetAccount returns the signed-in user's self-view, including API key fields.
func (s *Service) GetAccount(ctx context.Context, id int64) (*Account, error) {
	return s.store.GetAccountByID(ctx, id)
}

// UpdateAccount updates fields the signed-in user can manage for themselves.
func (s *Service) UpdateAccount(ctx context.Context, id int64, params AccountUpdateParams) (*Account, error) {
	if id == initialUserID {
		current, err := s.store.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}
		if params.Name != current.Name {
			return nil, ErrCannotModifyInitialUser
		}
	}
	return s.store.UpdateAccount(ctx, id, params)
}
