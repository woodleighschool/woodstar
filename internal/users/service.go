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

// InitialUserID is the row created by the setup wizard. That account is
// pinned as a permanent local password login: it cannot be deleted and only
// its password may be updated, so an admin always has a working login path.
const InitialUserID int64 = 1

// Service owns local Woodstar account management.
type Service struct {
	store *Store
}

// NewService returns a local user management service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// CreateParams contains fields needed to create a user.
type CreateParams struct {
	Email    string
	Name     string
	Role     Role
	Password string
}

// UpdateParams replaces the writable fields of a user.
type UpdateParams struct {
	Name     string
	Role     Role
	Password *string
}

// Exists reports whether any active local user exists.
func (s *Service) Exists(ctx context.Context) (bool, error) {
	return s.store.Exists(ctx)
}

// GetByEmail returns one active user by normalized email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetByEmail(ctx, email)
}

// Get returns one active user by id.
func (s *Service) Get(ctx context.Context, id int64) (*User, error) {
	return s.store.GetByID(ctx, id)
}

// List returns every active user.
func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.store.List(ctx)
}

// Create provisions a local user.
func (s *Service) Create(ctx context.Context, params CreateParams) (*User, error) {
	hash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	return s.store.Create(ctx, CreateRecordParams{
		Email:        params.Email,
		Name:         params.Name,
		PasswordHash: hash,
		Role:         params.Role,
	})
}

// Update writes the full target record. The initial user's name and role are
// locked.
func (s *Service) Update(ctx context.Context, targetID int64, params UpdateParams) (*User, error) {
	if targetID == InitialUserID {
		current, err := s.store.GetByID(ctx, targetID)
		if err != nil {
			return nil, err
		}
		if params.Name != current.Name || params.Role != current.Role {
			return nil, ErrCannotModifyInitialUser
		}
	}

	storeParams := UpdateRecordParams{
		Name: params.Name,
		Role: params.Role,
	}
	if params.Password != nil {
		hash, err := HashPassword(*params.Password)
		if err != nil {
			return nil, err
		}
		storeParams.PasswordHash = new(hash)
	}

	return s.store.Update(ctx, targetID, storeParams)
}

// Delete hard-deletes targetID. The initial user is protected so a working
// admin login always exists; the immutable id:1 admin floor also makes
// "last admin removed" structurally impossible.
func (s *Service) Delete(ctx context.Context, targetID int64) error {
	if targetID == InitialUserID {
		return ErrCannotDeleteInitialUser
	}
	return s.store.Delete(ctx, targetID)
}

// GetByAPIKey returns the user owning the given API key, or ErrNotFound.
func (s *Service) GetByAPIKey(ctx context.Context, key string) (*User, error) {
	return s.store.GetByAPIKey(ctx, key)
}

// SetAPIKey stores key as the API key for userID, replacing any prior key.
func (s *Service) SetAPIKey(ctx context.Context, userID int64, key string) (*Account, error) {
	return s.store.SetAPIKey(ctx, userID, key)
}

// ClearAPIKey removes the API key for userID.
func (s *Service) ClearAPIKey(ctx context.Context, userID int64) (*Account, error) {
	return s.store.ClearAPIKey(ctx, userID)
}
