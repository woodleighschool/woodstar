package users

import (
	"context"
	"errors"
)

// User management errors describe expected admin failures. The frontend
// gates self-mutation and last-admin removal in the UI; the backend only
// enforces invariants that protect the system from being locked out.
var (
	ErrCannotDeleteInitialUser = errors.New("the initial user cannot be deleted")
	ErrCannotModifyInitialUser = errors.New(
		"the initial user's name and role are locked; only the password may be changed",
	)
)

// initialUserID is the row created by the setup wizard.
const initialUserID int64 = 1

// Service owns local Woodstar account management.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Exists(ctx context.Context) (bool, error) {
	return s.store.Exists(ctx)
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetByEmail(ctx, email)
}

func (s *Service) Get(ctx context.Context, id int64) (*User, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.store.List(ctx)
}

// IsInitialUser reports whether user is the permanent local password login.
func (s *Service) IsInitialUser(user *User) bool {
	return user != nil && user.ID == initialUserID
}

func (s *Service) Create(ctx context.Context, params CreateParams) (*User, error) {
	return s.store.Create(ctx, params)
}

// Update writes the full target record. The initial user's name and role are
// locked.
func (s *Service) Update(ctx context.Context, targetID int64, params UpdateParams) (*User, error) {
	if targetID == initialUserID {
		current, err := s.store.GetByID(ctx, targetID)
		if err != nil {
			return nil, err
		}
		if s.IsInitialUser(current) && (params.Name != current.Name || params.Role != current.Role) {
			return nil, ErrCannotModifyInitialUser
		}
	}

	return s.store.Update(ctx, targetID, params)
}

// Delete hard-deletes targetID. The initial user is protected so a working
// admin login always exists; the immutable id:1 admin floor also makes
// "last admin removed" structurally impossible.
func (s *Service) Delete(ctx context.Context, targetID int64) error {
	if s.IsInitialUser(&User{ID: targetID}) {
		return ErrCannotDeleteInitialUser
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
