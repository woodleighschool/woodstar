package software

import (
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Mutation is the input shape for creating or updating Munki software.
type Mutation struct {
	Name         string  `json:"name"                     minLength:"1"`
	Description  string  `json:"description,omitempty"`
	Category     string  `json:"category,omitempty"`
	Developer    string  `json:"developer,omitempty"`
	IconObjectID *int64  `json:"icon_object_id,omitempty"`
	Targets      Targets `json:"targets"                                nullable:"false"`
}

func (m Mutation) validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	return nil
}

// Software is Woodstar-managed metadata for a Munki software item.
type Software struct {
	ID           int64     `json:"id"                       db:"id"`
	Name         string    `json:"name"                     db:"name"`
	Description  string    `json:"description"              db:"description"`
	Category     string    `json:"category"                 db:"category"`
	Developer    string    `json:"developer"                db:"developer"`
	IconObjectID *int64    `json:"icon_object_id,omitempty" db:"icon_object_id"`
	CreatedAt    time.Time `json:"created_at"               db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"               db:"updated_at"`
}
