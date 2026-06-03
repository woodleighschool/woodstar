package users

import (
	"context"
)

// Service owns user management.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Exists(ctx context.Context) (bool, error) {
	return s.store.Exists(ctx)
}

func (s *Service) GetLoginByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetLoginByEmail(ctx, email)
}

func (s *Service) GetSSOByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetSSOByEmail(ctx, email)
}

func (s *Service) Get(ctx context.Context, id int64) (*User, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.store.List(ctx)
}

func (s *Service) Create(ctx context.Context, params UserCreate) (*User, error) {
	return s.store.Create(ctx, params)
}

// Update writes the full target record.
func (s *Service) Update(ctx context.Context, targetID int64, params UserMutation) (*User, error) {
	return s.store.Update(ctx, targetID, params)
}

// Delete removes local users and deactivates synced users.
func (s *Service) Delete(ctx context.Context, targetID int64) error {
	user, err := s.store.GetByID(ctx, targetID)
	if err != nil {
		return err
	}
	if user.Synced {
		return s.store.DeactivateSynced(ctx, targetID)
	}
	return s.store.Delete(ctx, targetID)
}

func (s *Service) GetByAPIKey(ctx context.Context, key string) (*User, error) {
	return s.store.GetByAPIKey(ctx, key)
}

func (s *Service) SetAPIKey(ctx context.Context, userID int64, key string) (*Account, error) {
	return s.store.SetAPIKey(ctx, userID, key)
}

func (s *Service) ClearAPIKey(ctx context.Context, userID int64) (*Account, error) {
	return s.store.ClearAPIKey(ctx, userID)
}
