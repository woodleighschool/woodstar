package labels

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// LabelType marks builtin vs regular labels.
type LabelType string

const (
	LabelTypeBuiltin LabelType = "builtin"
	LabelTypeRegular LabelType = "regular"
)

// Label membership types.
const (
	LabelMembershipTypeDynamic = "dynamic"
	LabelMembershipTypeManual  = "manual"
	LabelMembershipTypeDerived = "derived"
)

// Label groups hosts.
type Label struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	Query               *string   `json:"query,omitempty"`
	LabelType           LabelType `json:"label_type"`
	LabelMembershipType string    `json:"label_membership_type"`
	HostsCount          int       `json:"hosts_count"`
	CreatedAt           time.Time `json:"created_at,omitzero"`
	UpdatedAt           time.Time `json:"updated_at,omitzero"`
}

// ListParams filters labels.
type ListParams struct {
	dbutil.ListParams

	LabelType           LabelType
	LabelMembershipType string
}

// LabelCreate is a new label.
type LabelCreate struct {
	Name                string
	Description         string
	Query               *string
	LabelType           LabelType
	LabelMembershipType string
}

// LabelUpdate is the editable label state.
type LabelUpdate struct {
	Name                string
	Description         string
	Query               *string
	LabelMembershipType string
}
