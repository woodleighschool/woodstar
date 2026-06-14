package software

import "time"

// Mutation is the input shape for creating or updating Munki software.
type Mutation struct {
	Name         string  `json:"name"                     minLength:"1"`
	Description  string  `json:"description,omitempty"`
	Category     string  `json:"category,omitempty"`
	Developer    string  `json:"developer,omitempty"`
	IconObjectID *int64  `json:"icon_object_id,omitempty"`
	Targets      Targets `json:"targets"                                nullable:"false"`
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
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
