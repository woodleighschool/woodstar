package software

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/munki/assignments"
)

// SoftwareTitleMutation is the input shape for creating or updating a Munki software title.
type SoftwareTitleMutation struct {
	Name            string                                  `json:"name"                        minLength:"1"`
	Description     string                                  `json:"description,omitempty"`
	Category        string                                  `json:"category,omitempty"`
	Developer       string                                  `json:"developer,omitempty"`
	IconName        string                                  `json:"icon_name,omitempty"`
	IconHash        string                                  `json:"icon_hash,omitempty"`
	IconArtifactID  *int64                                  `json:"icon_artifact_id,omitempty"`
	Includes        []assignments.AssignmentIncludeMutation `json:"includes,omitempty"`
	ExcludeLabelIDs []int64                                 `json:"exclude_label_ids,omitempty"`
}

// SoftwareTitle is Woodstar-managed metadata for a Munki software item.
type SoftwareTitle struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
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
