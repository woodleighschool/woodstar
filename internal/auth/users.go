package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/models"
)

// User management errors describe expected admin failures.
var (
	ErrCannotChangeOwnRole   = errors.New("cannot change your own role")
	ErrCannotDeleteSelf      = errors.New("cannot delete your own account")
	ErrCannotRemoveLastAdmin = errors.New("cannot remove the last admin")
)

// CreateUserParams contains fields needed to create a user.
type CreateUserParams struct {
	Email    string
	Name     string
	Role     models.UserRole
	Password string
}

// UpdateUserParams replaces the writable fields of a user.
type UpdateUserParams struct {
	Name     string
	Role     models.UserRole
	Password *string
}

// ListUsers returns every active user.
func (s *Service) ListUsers(ctx context.Context) ([]models.User, error) {
	if s.users == nil {
		return nil, ErrNotSetup
	}
	return s.users.List(ctx)
}

// GetUser returns one active user by id.
func (s *Service) GetUser(ctx context.Context, id int64) (*models.User, error) {
	if s.users == nil {
		return nil, ErrNotSetup
	}
	return s.users.GetByID(ctx, id)
}

// CreateUser provisions a local user.
func (s *Service) CreateUser(ctx context.Context, params CreateUserParams) (*models.User, error) {
	if s.users == nil {
		return nil, ErrNotSetup
	}

	hash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	return s.users.Create(ctx, models.CreateUserParams{
		Email:        params.Email,
		Name:         fallbackName(params.Name, params.Email),
		PasswordHash: hash,
		Role:         params.Role,
	})
}

// UpdateUser writes the full target record. Self-mutation guards.
func (s *Service) UpdateUser(
	ctx context.Context,
	actor *models.User,
	targetID int64,
	params UpdateUserParams,
) (*models.User, error) {
	if s.users == nil {
		return nil, ErrNotSetup
	}

	if actor != nil && actor.ID == targetID && actor.Role != params.Role {
		return nil, ErrCannotChangeOwnRole
	}

	if err := s.guardLastAdminOnRoleChange(ctx, targetID, params.Role); err != nil {
		return nil, err
	}

	storeParams := models.UpdateUserParams{
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

	return s.users.Update(ctx, targetID, storeParams)
}

// DeleteUser soft-deletes targetID. Active sessions in pgxstore are not removed
// here; CurrentUser denies them on the next request because users.GetByID
// filters deleted_at, and the rows expire on their own.
// actorID may not equal targetID, and the last active admin cannot be removed.
func (s *Service) DeleteUser(ctx context.Context, actorID int64, targetID int64) error {
	if s.users == nil {
		return ErrNotSetup
	}
	if actorID == targetID {
		return ErrCannotDeleteSelf
	}

	target, err := s.users.GetByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role == models.RoleAdmin {
		count, err := s.users.CountAdmins(ctx)
		if err != nil {
			return fmt.Errorf("count admins: %w", err)
		}
		if count <= 1 {
			return ErrCannotRemoveLastAdmin
		}
	}

	if err := s.users.SoftDelete(ctx, targetID); err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}
	return nil
}

// guardLastAdminOnRoleChange refuses a role change that would leave zero admins.
func (s *Service) guardLastAdminOnRoleChange(
	ctx context.Context,
	targetID int64,
	newRole models.UserRole,
) error {
	if newRole == models.RoleAdmin {
		return nil
	}
	target, err := s.users.GetByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role != models.RoleAdmin {
		return nil
	}
	count, err := s.users.CountAdmins(ctx)
	if err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if count <= 1 {
		return ErrCannotRemoveLastAdmin
	}
	return nil
}
