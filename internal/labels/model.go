package labels

import (
	"encoding/json"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

// BuiltinKey identifies a built-in label independent of display text.
type BuiltinKey string

type (
	// LabelType marks builtin vs regular labels.
	LabelType string
	// LabelMembershipType marks how hosts join a label: dynamic, manual, or derived.
	LabelMembershipType string
	// DerivedAttribute marks the source field used for derived label membership.
	DerivedAttribute string
)

const (
	LabelTypeBuiltin LabelType = "builtin"
	LabelTypeRegular LabelType = "regular"
)

// BuiltinKeyAllHosts identifies the built-in label matching every enrolled host.
const BuiltinKeyAllHosts BuiltinKey = "all-hosts"

var (
	builtinKeyValues = []BuiltinKey{BuiltinKeyAllHosts}
	LabelTypeValues  = []LabelType{LabelTypeBuiltin, LabelTypeRegular}
)

// Label membership types.
const (
	LabelMembershipTypeDynamic LabelMembershipType = "dynamic"
	LabelMembershipTypeManual  LabelMembershipType = "manual"
	LabelMembershipTypeDerived LabelMembershipType = "derived"
)

var LabelMembershipTypeValues = []LabelMembershipType{
	LabelMembershipTypeDynamic,
	LabelMembershipTypeManual,
	LabelMembershipTypeDerived,
}

// Derived label attributes.
const (
	DerivedAttributeUserDepartment DerivedAttribute = "user_department"
	DerivedAttributeDirectoryGroup DerivedAttribute = "directory_group"
	DerivedAttributeUser           DerivedAttribute = "user"
)

var DerivedAttributeValues = []DerivedAttribute{
	DerivedAttributeUserDepartment,
	DerivedAttributeDirectoryGroup,
	DerivedAttributeUser,
}

// Label groups hosts.
type Label struct {
	ID                  int64               `json:"id"`
	Name                string              `json:"name"`
	BuiltinKey          *BuiltinKey         `json:"builtin_key,omitempty" readOnly:"true"`
	Description         string              `json:"description"`
	Query               *string             `json:"query,omitempty"`
	Criteria            *Criteria           `json:"criteria,omitempty"`
	HostIDs             []int64             `json:"host_ids,omitempty"`
	LabelType           LabelType           `json:"label_type"`
	LabelMembershipType LabelMembershipType `json:"label_membership_type"`
	HostsCount          int32               `json:"hosts_count"`
	CreatedAt           time.Time           `json:"created_at,omitzero"`
	UpdatedAt           time.Time           `json:"updated_at,omitzero"`
}

// Criteria describes the non-osquery host attribute that derives membership.
type Criteria struct {
	Attribute DerivedAttribute `json:"attribute"`
	Values    []string         `json:"values"`
}

func (c Criteria) json() ([]byte, error) {
	return json.Marshal(c)
}

// LabelListParams filters labels.
type LabelListParams struct {
	dbutil.ListParams

	LabelType           LabelType
	LabelMembershipType LabelMembershipType
}

// LabelMutation is the editable label state used by create and update.
type LabelMutation struct {
	Name                string              `json:"name"`
	Description         string              `json:"description,omitempty"`
	Query               *string             `json:"query,omitempty"`
	Criteria            *Criteria           `json:"criteria,omitempty"`
	HostIDs             []int64             `json:"host_ids,omitempty"`
	LabelMembershipType LabelMembershipType `json:"label_membership_type,omitempty"`
}

func (BuiltinKey) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(builtinKeyValues...)
}

func (LabelType) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(LabelTypeValues...)
}

func (LabelMembershipType) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(LabelMembershipTypeValues...)
}

func (DerivedAttribute) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(DerivedAttributeValues...)
}
