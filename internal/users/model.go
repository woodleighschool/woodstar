package users

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// Role controls application permissions.
type Role = sqlc.UserRole

// User is a local Woodstar account.
type User struct {
	ID              int64      `json:"id"`
	Email           string     `json:"email"            format:"email"`
	Name            string     `json:"name"`
	PasswordHash    string     `json:"-"`
	Role            Role       `json:"role"                            enum:"admin,viewer"`
	APIKey          string     `json:"-"`
	APIKeyCreatedAt *time.Time `json:"api_key_created_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// HasAPIKey reports whether the user has an API key currently set.
func (u User) HasAPIKey() bool {
	return u.APIKey != ""
}
