package directory

import (
	"context"
)

// UserService owns user management and app-access policy.
type UserService struct {
	store *Store
}

func NewUserService(store *Store) *UserService {
	return &UserService{store: store}
}

func (s *UserService) Exists(ctx context.Context) (bool, error) {
	return s.store.UserExists(ctx)
}

func (s *UserService) GetLoginByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetLoginUserByEmail(ctx, email)
}

func (s *UserService) GetSSOByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetSSOUserByEmail(ctx, email)
}

func (s *UserService) Get(ctx context.Context, id int64) (*User, error) {
	return s.store.GetUserByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, params UserListParams) ([]User, int, error) {
	return s.store.ListUsers(ctx, params)
}

func (s *UserService) ListDepartments(ctx context.Context, params UserListParams) ([]Department, int, error) {
	return s.store.ListDepartments(ctx, params)
}

func (s *UserService) Create(ctx context.Context, params UserCreate) (*User, error) {
	return s.store.CreateUser(ctx, params)
}

// Update writes the full target record.
func (s *UserService) Update(ctx context.Context, targetID int64, params UserMutation) (*User, error) {
	return s.store.UpdateUser(ctx, targetID, params)
}

// Delete hard-deletes local users and soft-deletes source-owned identities.
func (s *UserService) Delete(ctx context.Context, targetID int64) error {
	user, err := s.store.GetUserByID(ctx, targetID)
	if err != nil {
		return err
	}
	if user.Source != SourceLocal {
		return s.store.SoftDeleteUser(ctx, targetID)
	}
	return s.store.DeleteUser(ctx, targetID)
}

func (s *UserService) GetByAPIKey(ctx context.Context, key string) (*User, error) {
	return s.store.GetUserByAPIKey(ctx, key)
}

func (s *UserService) SetAPIKey(ctx context.Context, userID int64, key string) (*Account, error) {
	return s.store.SetUserAPIKey(ctx, userID, key)
}

func (s *UserService) ClearAPIKey(ctx context.Context, userID int64) (*Account, error) {
	return s.store.ClearUserAPIKey(ctx, userID)
}
