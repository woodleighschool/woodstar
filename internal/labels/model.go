package labels

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/platforms"
)

// Label types. LabelType separates system-seeded labels from admin-created ones.
const (
	LabelTypeBuiltin = "builtin"
	LabelTypeRegular = "regular"
)

// Label membership types. LabelMembershipType controls how membership rows are produced:
//   - dynamic: an osquery query result drives membership
//   - manual: the server writes membership rows (e.g. All Hosts on enroll)
//   - derived: membership is computed from non-osquery host attributes (criteria JSON)
const (
	LabelMembershipTypeDynamic = "dynamic"
	LabelMembershipTypeManual  = "manual"
	LabelMembershipTypeDerived = "derived"
)

// Label is a host grouping and targeting primitive.
type Label struct {
	ID                  int64                `json:"id"`
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	Query               *string              `json:"query,omitempty"`
	LabelType           string               `json:"label_type"`
	LabelMembershipType string               `json:"label_membership_type"`
	Platforms           []platforms.Platform `json:"platforms"             minItems:"1" nullable:"false"`
	HostsCount          int                  `json:"hosts_count"`
	CreatedAt           time.Time            `json:"created_at,omitzero"`
	UpdatedAt           time.Time            `json:"updated_at,omitzero"`
}

// LabelListParams filters the admin label list.
type LabelListParams struct {
	dbutil.ListParams

	LabelType           string
	LabelMembershipType string
	Platform            string
}

// LabelCreate contains fields for an admin-created label.
type LabelCreate struct {
	Name                string
	Description         string
	Query               *string
	LabelType           string
	LabelMembershipType string
	Platforms           []platforms.Platform
}

// LabelUpdate contains the full editable label state.
type LabelUpdate struct {
	Name                string
	Description         string
	Query               *string
	LabelMembershipType string
	Platforms           []platforms.Platform
}
