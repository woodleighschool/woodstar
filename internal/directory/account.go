package directory

import (
	"context"
)

// AccountMutation contains fields a signed-in user can mutate on their own account.
type AccountMutation struct {
	Name     string  `json:"name"`
	Password *string `json:"password,omitempty"`
}

type accountUpdateRecord struct {
	Name         string
	PasswordHash *string
}

// GetAccount returns the signed-in user's self-view, including API key fields.
func (s *UserService) GetAccount(ctx context.Context, id int64) (*Account, error) {
	return s.store.GetAccountByID(ctx, id)
}

// UpdateAccount updates fields the signed-in user can manage for themselves.
func (s *UserService) UpdateAccount(ctx context.Context, id int64, params AccountMutation) (*Account, error) {
	passwordHash, err := hashOptionalPassword(params.Password)
	if err != nil {
		return nil, err
	}
	return s.store.updateAccount(ctx, id, accountUpdateRecord{
		Name:         params.Name,
		PasswordHash: passwordHash,
	})
}
