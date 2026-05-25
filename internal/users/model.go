package users

import (
	"time"
)

// Role controls permissions.
type Role string

// User roles.
const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

// User is a local account.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"      format:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"                      enum:"admin,viewer"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Account is the signed-in user's self-view, including their retrievable API key.
type Account struct {
	User            User
	APIKey          string
	APIKeyCreatedAt *time.Time
}
