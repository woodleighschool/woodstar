package directory

import (
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
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

// Department is one non-empty department observed on directory directory.
type Department struct {
	Value string `json:"value"`
}

// UserListParams filters paginated user lists.
type UserListParams struct {
	dbutil.ListParams

	Values  []string
	Role    string
	Source  string
	GroupID int64
}

func (Role) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(RoleValues...)
}

// Account is the signed-in user's self-view, including their retrievable API key.
type Account struct {
	User            User       `json:"user"`
	APIKey          string     `json:"api_key,omitempty"`
	APIKeyCreatedAt *time.Time `json:"api_key_created_at,omitempty"`
}
