package software

import "time"

// Mutation is the input shape for creating or updating Munki software.
type Mutation struct {
	Name           string  `json:"name"                       minLength:"1"`
	Description    string  `json:"description,omitempty"`
	Category       string  `json:"category,omitempty"`
	Developer      string  `json:"developer,omitempty"`
	IconName       string  `json:"icon_name,omitempty"`
	IconHash       string  `json:"icon_hash,omitempty"`
	IconArtifactID *int64  `json:"icon_artifact_id,omitempty"`
	Targets        Targets `json:"targets"                                  nullable:"false"`
}

// Software is Woodstar-managed metadata for a Munki software item.
type Software struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	Description          string    `json:"description"`
	Category             string    `json:"category"`
	Developer            string    `json:"developer"`
	IconName             string    `json:"icon_name"`
	IconHash             string    `json:"icon_hash"`
	IconArtifactID       *int64    `json:"icon_artifact_id,omitempty"`
	IconArtifactLocation string    `json:"icon_artifact_location,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
