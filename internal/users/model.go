package users

import (
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

// Role controls permissions.
type Role string

// User roles.
const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

var RoleValues = []Role{RoleAdmin, RoleViewer}

// User is a Woodstar person row, optionally granted app access by Role.
type User struct {
	ID                int64      `json:"id"`
	Email             string     `json:"email"                         format:"email"`
	Name              string     `json:"name"`
	PasswordHash      string     `json:"-"`
	Role              *Role      `json:"role,omitempty"`
	EntraID           string     `json:"entra_id,omitempty"`
	UserPrincipalName string     `json:"user_principal_name,omitempty"`
	MailNickname      string     `json:"mail_nickname,omitempty"`
	GivenName         string     `json:"given_name,omitempty"`
	FamilyName        string     `json:"family_name,omitempty"`
	Department        string     `json:"department,omitempty"`
	Active            bool       `json:"active"`
	Synced            bool       `json:"synced"`
	CanLogin          bool       `json:"can_login"`
	LastSyncedAt      *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Department is one non-empty department observed on synced users.
type Department struct {
	Value string `json:"value"`
}

// ListParams filters paginated user lists.
type ListParams struct {
	dbutil.ListParams

	Values []string
	Role   string
	Source string
	Status string
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
