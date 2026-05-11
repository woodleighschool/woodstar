package labels

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/platform"
)

// Label is a host grouping and targeting primitive.
type Label struct {
	ID                  int64              `json:"id"`
	Name                string             `json:"name"`
	Description         string             `json:"description"`
	Query               *string            `json:"query"`
	LabelType           string             `json:"label_type"`
	LabelMembershipType string             `json:"label_membership_type"`
	Platform            *platform.Platform `json:"platform"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`

	// HostsCount is populated by list/get queries via JOIN.
	HostsCount int
}
