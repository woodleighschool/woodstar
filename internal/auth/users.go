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

// UpdateUserParams contains the optional fields an admin can change on a user.
// Nil fields are left untouched.
type UpdateUserParams struct {
	Name     *string
	Role     *models.UserRole
	Password *string
}

// ListUsers returns every active user.
func (s *Service) ListUsers(ctx context.Context) ([]models.User, error) {
	if s.users == nil {
		return nil, ErrNotSetup
	}
	return s.users.List(ctx)
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

// UpdateUser applies the non-nil fields of params to targetID.
// actorID is the admin making the change and is used to enforce self-mutation guards.
func (s *Service) UpdateUser(
	ctx context.Context,
	actorID int64,
	targetID int64,
	params UpdateUserParams,
) (*models.User, error) {
	if s.users == nil {
		return nil, ErrNotSetup
	}

	if actorID == targetID && params.Role != nil {
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

// DeleteUser soft-deletes targetID and revokes all of their sessions.
// actorID may not equal targetID, and the last active admin cannot be removed.
func (s *Service) DeleteUser(ctx context.Context, actorID int64, targetID int64) error {
	if s.users == nil || s.sessions == nil {
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

	if err := s.sessions.RevokeAllForUser(ctx, targetID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
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
	newRole *models.UserRole,
) error {
	if newRole == nil || *newRole == models.RoleAdmin {
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
