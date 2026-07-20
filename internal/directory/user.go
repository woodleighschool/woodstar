package directory

import (
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// Role controls permissions.
type (
	Role string
)

// User roles.
const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

var RoleValues = []Role{RoleAdmin, RoleViewer}

// User is a Woodstar account row, optionally granted app access by Role.
type User struct {
	ID                int64      `json:"id"`
	Email             string     `json:"email"                         format:"email"`
	Name              string     `json:"name"`
	PasswordHash      string     `json:"-"`
	Role              *Role      `json:"role,omitempty"`
	Source            Source     `json:"source"`
	ExternalID        string     `json:"external_id,omitempty"`
	UserPrincipalName string     `json:"user_principal_name,omitempty"`
	MailNickname      string     `json:"mail_nickname,omitempty"`
	GivenName         string     `json:"given_name,omitempty"`
	FamilyName        string     `json:"family_name,omitempty"`
	Department        string     `json:"department,omitempty"`
	CanLogin          bool       `json:"can_login"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Department is one non-empty department value drawn from directory users.
type Department struct {
	Value string `json:"value"`
}

// UserListParams filters paginated user lists.
type UserListParams struct {
	dbutil.ListParams

	Values  []string
	Role    string `validate:"omitempty,oneof=admin viewer none"`
	Source  string `validate:"omitempty,oneof=local entra"`
	GroupID int64  `validate:"gte=0"`
}

// UserCreate contains fields needed to create a user.
type UserCreate struct {
	Email    string `json:"email"          format:"email" validate:"required,email"`
	Name     string `json:"name,omitempty"`
	Role     Role   `json:"role"                          validate:"required,oneof=admin viewer"`
	Password string `json:"password"                                                             minLength:"12"`
}

// UserMutation replaces the writable fields of a user.
type UserMutation struct {
	Name     string  `json:"name"`
	Role     *Role   `json:"role,omitempty"     validate:"omitempty,oneof=admin viewer"`
	Password *string `json:"password,omitempty"`
}

func (params *UserListParams) normalize() {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	params.Values = dbutil.NormalizeListValues(params.Values)
	params.Role = strings.TrimSpace(params.Role)
	params.Source = strings.TrimSpace(params.Source)
}

func (params *UserListParams) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (params *UserCreate) normalize() {
	params.Email = strings.TrimSpace(params.Email)
	params.Name = strings.TrimSpace(params.Name)
	params.Role = Role(strings.TrimSpace(string(params.Role)))
}

func (params *UserCreate) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (params *UserMutation) normalize() {
	params.Name = strings.TrimSpace(params.Name)
	if params.Role != nil {
		role := Role(strings.TrimSpace(string(*params.Role)))
		params.Role = &role
	}
}

func (params *UserMutation) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (Role) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(RoleValues...)
}

// Account is the signed-in user's self-view, including their retrievable API key.
type Account struct {
	User            User       `json:"user"`
	APIKey          string     `json:"api_key,omitempty"`
	APIKeyCreatedAt *time.Time `json:"api_key_created_at,omitempty"`
}
