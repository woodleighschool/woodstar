package users

import (
	"time"

	"github.com/danielgtaylor/huma/v2"

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

// User is a local account.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"      format:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Role) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(RoleValues...)
}

// Account is the signed-in user's self-view, including their retrievable API key.
type Account struct {
	User            User
	APIKey          string
	APIKeyCreatedAt *time.Time
}
