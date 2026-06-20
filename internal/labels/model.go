package labels

import (
	"fmt"
	"strings"
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

// Validate checks the label shape before the DB sees it.
func (p LabelMutation) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	return validateMembershipPairing(p.LabelMembershipType, p.Query, p.Criteria, p.HostIDs)
}

func validateMembershipPairing(
	membershipType LabelMembershipType,
	query *string,
	criteria *Criteria,
	hostIDs []int64,
) error {
	switch membershipType {
	case LabelMembershipTypeDynamic:
		return validateDynamicMembership(query, criteria, hostIDs)
	case LabelMembershipTypeManual:
		return validateManualMembership(query, criteria)
	case LabelMembershipTypeDerived:
		return validateDerivedMembership(query, criteria, hostIDs)
	default:
		return fmt.Errorf("%w: membership type must be dynamic, manual, or derived", dbutil.ErrInvalidInput)
	}
}

func validateDynamicMembership(query *string, criteria *Criteria, hostIDs []int64) error {
	if query == nil || strings.TrimSpace(*query) == "" {
		return fmt.Errorf("%w: query is required for dynamic labels", dbutil.ErrInvalidInput)
	}
	if criteria != nil {
		return fmt.Errorf("%w: criteria is only allowed for derived labels", dbutil.ErrInvalidInput)
	}
	if len(hostIDs) > 0 {
		return fmt.Errorf("%w: hosts are only allowed for manual labels", dbutil.ErrInvalidInput)
	}
	return nil
}

func validateManualMembership(query *string, criteria *Criteria) error {
	if query != nil {
		return fmt.Errorf("%w: query is only allowed for dynamic labels", dbutil.ErrInvalidInput)
	}
	if criteria != nil {
		return fmt.Errorf("%w: criteria is only allowed for derived labels", dbutil.ErrInvalidInput)
	}
	return nil
}

func validateDerivedMembership(query *string, criteria *Criteria, hostIDs []int64) error {
	if query != nil {
		return fmt.Errorf("%w: query is only allowed for dynamic labels", dbutil.ErrInvalidInput)
	}
	if len(hostIDs) > 0 {
		return fmt.Errorf("%w: hosts are only allowed for manual labels", dbutil.ErrInvalidInput)
	}
	return validateCriteria(criteria)
}

func validateCriteria(criteria *Criteria) error {
	if criteria == nil {
		return fmt.Errorf("%w: criteria is required for derived labels", dbutil.ErrInvalidInput)
	}
	switch criteria.Attribute {
	case DerivedAttributeUserDepartment, DerivedAttributeDirectoryGroup, DerivedAttributeUser:
	default:
		return fmt.Errorf("%w: unknown derived label attribute", dbutil.ErrInvalidInput)
	}
	if len(cleanCriteriaValues(criteria.Values)) == 0 {
		return fmt.Errorf("%w: derived label values are required", dbutil.ErrInvalidInput)
	}
	return nil
}

func cleanCriteriaValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
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
