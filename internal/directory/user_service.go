package directory

import (
	"context"
	"errors"
)

var (
	// ErrSetupComplete is returned when initial setup has already completed.
	ErrSetupComplete = errors.New("initial setup is complete")
	// ErrLastAdministrator is returned when a mutation would remove the final active administrator.
	ErrLastAdministrator = errors.New("at least one active administrator is required")
)

// UserService owns user management and app-access policy.
type UserService struct {
	store *Store
}

// NewUserService returns the user-management service.
func NewUserService(store *Store) *UserService {
	return &UserService{store: store}
}

// ActiveAdministratorExists reports whether a current user has the administrator role.
func (s *UserService) ActiveAdministratorExists(ctx context.Context) (bool, error) {
	return s.store.ActiveAdministratorExists(ctx)
}

// SetupComplete reports whether initial setup has completed.
func (s *UserService) SetupComplete(ctx context.Context) (bool, error) {
	return s.store.SetupComplete(ctx)
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
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	return s.store.ListUsers(ctx, params)
}

func (s *UserService) ListDepartments(ctx context.Context, params UserListParams) ([]Department, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	return s.store.ListDepartments(ctx, params)
}

func (s *UserService) Create(ctx context.Context, params UserCreate) (*User, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, err
	}
	hash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}
	return s.store.createUser(ctx, userCreateRecord{
		Email:        params.Email,
		Name:         params.Name,
		PasswordHash: hash,
		Role:         params.Role,
	})
}

// CreateInitialAdministrator completes setup with a local administrator.
func (s *UserService) CreateInitialAdministrator(ctx context.Context, params UserCreate) (*User, error) {
	params.Role = RoleAdmin
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, err
	}
	complete, err := s.store.SetupComplete(ctx)
	if err != nil {
		return nil, err
	}
	if complete {
		return nil, ErrSetupComplete
	}
	hash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}
	return s.store.createInitialAdministrator(ctx, initialAdministratorRecord{
		Email:        params.Email,
		Name:         params.Name,
		PasswordHash: hash,
	})
}

// Update writes the full target record.
func (s *UserService) Update(ctx context.Context, targetID int64, params UserMutation) (*User, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, err
	}
	passwordHash, err := hashOptionalPassword(params.Password)
	if err != nil {
		return nil, err
	}
	return s.store.updateUser(ctx, targetID, userUpdateRecord{
		Name:         params.Name,
		Role:         params.Role,
		PasswordHash: passwordHash,
	})
}

// Delete hard-deletes local users and soft-deletes source-owned identities.
func (s *UserService) Delete(ctx context.Context, targetID int64) error {
	return s.store.deleteUser(ctx, targetID)
}

func (s *UserService) GetByAPIKey(ctx context.Context, key string) (*User, error) {
	return s.store.GetUserByAPIKey(ctx, key)
}

// SetAccountAPIKey writes a generated API key to the user's account record.
func (s *UserService) SetAccountAPIKey(ctx context.Context, userID int64, key string) (*Account, error) {
	return s.store.setAccountAPIKey(ctx, userID, key)
}

// ClearAccountAPIKey removes the user's account API key.
func (s *UserService) ClearAccountAPIKey(ctx context.Context, userID int64) (*Account, error) {
	return s.store.clearAccountAPIKey(ctx, userID)
}

func hashOptionalPassword(password *string) (*string, error) {
	if password == nil {
		return nil, nil
	}
	hash, err := HashPassword(*password)
	if err != nil {
		return nil, err
	}
	return &hash, nil
}
