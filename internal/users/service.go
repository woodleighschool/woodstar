package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// User management errors describe expected admin failures.
var (
	ErrCannotChangeOwnRole   = errors.New("cannot change your own role")
	ErrCannotDeleteSelf      = errors.New("cannot delete your own account")
	ErrCannotRemoveLastAdmin = errors.New("cannot remove the last admin")
)

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
		Name:         fallbackName(params.Name, params.Email),
		PasswordHash: hash,
		Role:         params.Role,
	})
}

// Update writes the full target record. Self-mutation guards.
func (s *Service) Update(
	ctx context.Context,
	actor *User,
	targetID int64,
	params UpdateParams,
) (*User, error) {
	if actor != nil && actor.ID == targetID && actor.Role != params.Role {
		return nil, ErrCannotChangeOwnRole
	}

	if err := s.guardLastAdminOnRoleChange(ctx, targetID, params.Role); err != nil {
		return nil, err
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
		storeParams.PasswordHash = &hash
	}

	return s.store.Update(ctx, targetID, storeParams)
}

// Delete soft-deletes targetID. Active sessions in pgxstore are not removed
// here; auth.CurrentUser denies them on the next request because GetByID
// filters deleted_at, and the rows expire on their own.
// actorID may not equal targetID, and the last active admin cannot be removed.
func (s *Service) Delete(ctx context.Context, actorID int64, targetID int64) error {
	if actorID == targetID {
		return ErrCannotDeleteSelf
	}

	target, err := s.store.GetByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role == RoleAdmin {
		count, err := s.store.CountAdmins(ctx)
		if err != nil {
			return fmt.Errorf("count admins: %w", err)
		}
		if count <= 1 {
			return ErrCannotRemoveLastAdmin
		}
	}

	if err := s.store.SoftDelete(ctx, targetID); err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}
	return nil
}

// guardLastAdminOnRoleChange refuses a role change that would leave zero admins.
func (s *Service) guardLastAdminOnRoleChange(
	ctx context.Context,
	targetID int64,
	newRole Role,
) error {
	if newRole == RoleAdmin {
		return nil
	}
	target, err := s.store.GetByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role != RoleAdmin {
		return nil
	}
	count, err := s.store.CountAdmins(ctx)
	if err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if count <= 1 {
		return ErrCannotRemoveLastAdmin
	}
	return nil
}

func fallbackName(name string, email string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(email)
}
