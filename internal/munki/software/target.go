package software

import (
	"fmt"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// SoftwareTargets is the include/exclude label targeting contract for Munki software.
type SoftwareTargets struct {
	Include []SoftwareInclude    `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

// SoftwareInclude applies desired Munki package state to hosts with a matching label.
type SoftwareInclude struct {
	LabelID  int64                   `json:"label_id" minimum:"1"`
	Package  SoftwarePackageSelector `json:"package"`
	State    SoftwareState           `json:"state"`
	Featured bool                    `json:"featured"`
}

// SoftwarePackageSelector chooses the package candidate set for a software include.
type SoftwarePackageSelector struct {
	Strategy  SoftwarePackageStrategy `json:"strategy"`
	PackageID *int64                  `json:"package_id,omitempty" minimum:"1"`
}

// SoftwarePackageStrategy describes whether Munki software follows the latest
// eligible package or pins one package version.
type SoftwarePackageStrategy string

const (
	SoftwarePackageLatest   SoftwarePackageStrategy = "latest"
	SoftwarePackageSpecific SoftwarePackageStrategy = "specific"
)

var softwarePackageStrategyValues = []SoftwarePackageStrategy{
	SoftwarePackageLatest,
	SoftwarePackageSpecific,
}

// SoftwareState describes the Munki manifest section for an include.
type SoftwareState string

const (
	SoftwareStateManagedInstall   SoftwareState = "managed_install"
	SoftwareStateManagedUninstall SoftwareState = "managed_uninstall"
	SoftwareStateManagedUpdate    SoftwareState = "managed_update"
	SoftwareStateOptionalInstall  SoftwareState = "optional_install"
)

var softwareStateValues = []SoftwareState{
	SoftwareStateManagedInstall,
	SoftwareStateManagedUninstall,
	SoftwareStateManagedUpdate,
	SoftwareStateOptionalInstall,
}

// EffectivePackage is a host-resolved Munki package ready for manifest/catalog rendering.
type EffectivePackage struct {
	TargetID   int64
	SoftwareID int64
	State      SoftwareState
	Package    packages.Package
	// SoftwareIcon is software-owned pkginfo context projected with the package.
	SoftwareIcon packages.IconRef
	Selector     SoftwarePackageSelector
	Featured     bool
}

// Schema returns the OpenAPI schema for SoftwarePackageStrategy.
func (SoftwarePackageStrategy) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(softwarePackageStrategyValues...)
}

// Schema returns the OpenAPI schema for SoftwareState.
func (SoftwareState) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(softwareStateValues...)
}

func normalizeSoftwareTargets(targets SoftwareTargets) SoftwareTargets {
	if targets.Include == nil {
		targets.Include = []SoftwareInclude{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []targeting.LabelRef{}
	}
	for i := range targets.Include {
		targets.Include[i].Package.Strategy = SoftwarePackageStrategy(
			strings.TrimSpace(string(targets.Include[i].Package.Strategy)),
		)
		targets.Include[i].State = SoftwareState(strings.TrimSpace(string(targets.Include[i].State)))
	}
	return targets
}

func emptySoftwareTargets() SoftwareTargets {
	return SoftwareTargets{
		Include: []SoftwareInclude{},
		Exclude: []targeting.LabelRef{},
	}
}

func (targets SoftwareTargets) validate() error {
	for _, include := range targets.Include {
		if err := include.validate(); err != nil {
			return err
		}
	}
	if err := targeting.ValidateTargets(targets.Include, targets.Exclude, softwareIncludeLabelID); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (include SoftwareInclude) validate() error {
	if !validSoftwarePackageStrategy(include.Package.Strategy) {
		return fmt.Errorf("%w: package.strategy is required", dbutil.ErrInvalidInput)
	}
	if !validSoftwareState(include.State) {
		return fmt.Errorf("%w: state is required", dbutil.ErrInvalidInput)
	}
	if err := include.Package.validate(); err != nil {
		return err
	}
	if include.Featured && include.State != SoftwareStateOptionalInstall {
		return fmt.Errorf("%w: featured requires optional_install state", dbutil.ErrInvalidInput)
	}
	return nil
}

func (selector SoftwarePackageSelector) validate() error {
	switch selector.Strategy {
	case SoftwarePackageLatest:
		if selector.PackageID != nil {
			return fmt.Errorf("%w: package.package_id must be empty for latest strategy", dbutil.ErrInvalidInput)
		}
	case SoftwarePackageSpecific:
		if selector.PackageID == nil {
			return fmt.Errorf("%w: package.package_id is required for specific strategy", dbutil.ErrInvalidInput)
		}
		if *selector.PackageID <= 0 {
			return fmt.Errorf("%w: package.package_id must be positive", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

func validSoftwarePackageStrategy(strategy SoftwarePackageStrategy) bool {
	return slices.Contains(softwarePackageStrategyValues, strategy)
}

func validSoftwareState(state SoftwareState) bool {
	return slices.Contains(softwareStateValues, state)
}

func softwareIncludeLabelID(include SoftwareInclude) int64 {
	return include.LabelID
}

func labelRefIDs(refs []targeting.LabelRef) []int64 {
	ids := make([]int64, len(refs))
	for i, ref := range refs {
		ids[i] = ref.LabelID
	}
	return ids
}
