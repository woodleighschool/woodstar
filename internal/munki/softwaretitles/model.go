package softwaretitles

import (
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// SoftwareTitleMutation is the input shape for creating or updating a Munki software title.
type SoftwareTitleMutation struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name,omitempty"`
	Description    string `json:"description,omitempty"`
	Category       string `json:"category,omitempty"`
	Developer      string `json:"developer,omitempty"`
	IconName       string `json:"icon_name,omitempty"`
	IconHash       string `json:"icon_hash,omitempty"`
	IconArtifactID *int64 `json:"icon_artifact_id,omitempty"`
}

// SoftwareTitle is Woodstar-managed metadata for a Munki software item.
type SoftwareTitle struct {
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

func (m SoftwareTitleMutation) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	return nil
}
