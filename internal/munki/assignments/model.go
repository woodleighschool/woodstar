package assignments

import (
	"fmt"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

// AssignmentAction describes the managed Munki manifest section for an assignment.
type AssignmentAction string

const (
	AssignmentActionInstall         AssignmentAction = "install"
	AssignmentActionRemove          AssignmentAction = "remove"
	AssignmentActionUpdateIfPresent AssignmentAction = "update_if_present"
	AssignmentActionNone            AssignmentAction = "none"
)

var actionValues = []AssignmentAction{
	AssignmentActionInstall,
	AssignmentActionRemove,
	AssignmentActionUpdateIfPresent,
	AssignmentActionNone,
}

func (AssignmentAction) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(actionValues...)
}

// PackageSelection describes whether an assignment follows the latest eligible
// package or pins one package version.
type PackageSelection string

const (
	PackageSelectionLatestEligible PackageSelection = "latest_eligible"
	PackageSelectionSpecific       PackageSelection = "specific_package"
)

var packageSelectionValues = []PackageSelection{
	PackageSelectionLatestEligible,
	PackageSelectionSpecific,
}

func (PackageSelection) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(packageSelectionValues...)
}

// AssignmentMutation is one ordered include label row for Munki desired state.
type AssignmentMutation struct {
	SoftwareID       int64            `json:"software_id"`
	Priority         int32            `json:"priority"`
	LabelID          int64            `json:"label_id"`
	Action           AssignmentAction `json:"action"`
	OptionalInstall  bool             `json:"optional_install,omitempty"`
	FeaturedItem     bool             `json:"featured_item,omitempty"`
	PackageSelection PackageSelection `json:"package_selection"`
	PinnedPackageID  *int64           `json:"pinned_package_id,omitempty"`
}

// Assignment links one Munki software title, one include label, and its Munki payload.
type Assignment struct {
	ID                   int64            `json:"id"`
	SoftwareID           int64            `json:"software_id"`
	SoftwareDisplayName  string           `json:"software_display_name"`
	Priority             int32            `json:"priority"`
	LabelID              int64            `json:"label_id"`
	Action               AssignmentAction `json:"action"`
	OptionalInstall      bool             `json:"optional_install"`
	FeaturedItem         bool             `json:"featured_item"`
	PackageSelection     PackageSelection `json:"package_selection"`
	PinnedPackageID      *int64           `json:"pinned_package_id,omitempty"`
	PinnedPackageName    string           `json:"pinned_package_name,omitempty"`
	PinnedPackageVersion string           `json:"pinned_package_version,omitempty"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

// EffectivePackage is a host-resolved Munki package ready for manifest/catalog rendering.
type EffectivePackage struct {
	AssignmentID     int64
	SoftwareID       int64
	Action           AssignmentAction
	OptionalInstall  bool
	FeaturedItem     bool
	PackageSelection PackageSelection
	PinnedPackageID  *int64
	Priority         int32
	Package          packages.Package
}

type AssignmentListParams struct {
	dbutil.ListParams
	SoftwareID int64
}

func (m AssignmentMutation) Validate() error {
	if m.SoftwareID <= 0 {
		return fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	if m.Priority < 1 {
		return fmt.Errorf("%w: priority must be at least 1", dbutil.ErrInvalidInput)
	}
	if !validAction(m.Action) {
		return fmt.Errorf("%w: unsupported assignment action %q", dbutil.ErrInvalidInput, m.Action)
	}
	if !validPackageSelection(m.PackageSelection) {
		return fmt.Errorf(
			"%w: unsupported package_selection %q",
			dbutil.ErrInvalidInput,
			m.PackageSelection,
		)
	}
	return m.validatePackagePayload()
}

func validAction(action AssignmentAction) bool {
	return slices.Contains(actionValues, action)
}

func validPackageSelection(selection PackageSelection) bool {
	return slices.Contains(packageSelectionValues, selection)
}

func (m AssignmentMutation) validatePackagePayload() error {
	switch m.PackageSelection {
	case PackageSelectionLatestEligible:
		if m.PinnedPackageID != nil {
			return fmt.Errorf(
				"%w: pinned_package_id must be empty for latest_eligible selection",
				dbutil.ErrInvalidInput,
			)
		}
	case PackageSelectionSpecific:
		if m.PinnedPackageID == nil {
			return fmt.Errorf("%w: pinned_package_id is required", dbutil.ErrInvalidInput)
		}
	}
	if m.FeaturedItem && !m.OptionalInstall {
		return fmt.Errorf("%w: featured_item requires optional_install", dbutil.ErrInvalidInput)
	}
	if m.Action == AssignmentActionRemove && (m.OptionalInstall || m.FeaturedItem) {
		return fmt.Errorf(
			"%w: remove assignments cannot be optional_installs or featured_items",
			dbutil.ErrInvalidInput,
		)
	}
	return nil
}
