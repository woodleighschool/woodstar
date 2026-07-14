package software

import (
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// Mutation is the input shape for creating or updating Munki software.
type Mutation struct {
	Name         string  `json:"name"                     minLength:"1" validate:"required,notblank"`
	DisplayName  string  `json:"display_name,omitempty"`
	Description  string  `json:"description,omitempty"`
	Category     string  `json:"category,omitempty"`
	Developer    string  `json:"developer,omitempty"`
	IconObjectID *int64  `json:"icon_object_id,omitempty"               validate:"omitempty,gt=0"    minimum:"1"`
	Targets      Targets `json:"targets"                                                                         nullable:"false"`
}

func (m *Mutation) validate() error {
	if err := validation.Struct(m); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (m *Mutation) normalize() {
	m.Name = strings.TrimSpace(m.Name)
	m.DisplayName = strings.TrimSpace(m.DisplayName)
	if m.DisplayName == "" {
		m.DisplayName = m.Name
	}
	m.Description = strings.TrimSpace(m.Description)
	m.Category = strings.TrimSpace(m.Category)
	m.Developer = strings.TrimSpace(m.Developer)
	m.Targets = normalizeTargets(m.Targets)
}

// Software is Woodstar-managed metadata for a Munki software item.
type Software struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	Category     string    `json:"category"`
	Developer    string    `json:"developer"`
	IconObjectID *int64    `json:"icon_object_id,omitempty"`
	IconFile     *IconFile `json:"icon_file,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IconFile describes the object attached as a software icon.
type IconFile struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}
