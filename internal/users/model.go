package users

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// Role controls application permissions.
type Role = sqlc.UserRole

// User is a local Woodstar account.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"      format:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"                      enum:"admin,viewer"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
