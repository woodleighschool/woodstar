package software

import (
	"fmt"
	"slices"

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

// SoftwareInclude applies desired Munki manifest actions to hosts with a matching label.
type SoftwareInclude struct {
	LabelID int64                   `json:"label_id" minimum:"1"`
	Package SoftwarePackageSelector `json:"package"`
	Actions []SoftwareAction        `json:"actions"              minItems:"1" nullable:"false"`
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

// SoftwareAction describes one Munki manifest section for an include.
type SoftwareAction string

const (
	SoftwareActionManagedInstalls   SoftwareAction = "managed_installs"
	SoftwareActionManagedUninstalls SoftwareAction = "managed_uninstalls"
	SoftwareActionManagedUpdates    SoftwareAction = "managed_updates"
	SoftwareActionOptionalInstalls  SoftwareAction = "optional_installs"
	SoftwareActionFeaturedItems     SoftwareAction = "featured_items"
	SoftwareActionDefaultInstalls   SoftwareAction = "default_installs"
)

var softwareActionValues = []SoftwareAction{
	SoftwareActionManagedInstalls,
	SoftwareActionManagedUninstalls,
	SoftwareActionManagedUpdates,
	SoftwareActionOptionalInstalls,
	SoftwareActionFeaturedItems,
	SoftwareActionDefaultInstalls,
}

// EffectivePackage is a host-resolved Munki package ready for manifest/catalog rendering.
type EffectivePackage struct {
	TargetID   int64
	SoftwareID int64
	Actions    []SoftwareAction
	Package    packages.Package
	// SoftwareIcon is software-owned pkginfo context projected with the package.
	SoftwareIcon packages.IconRef
	Selector     SoftwarePackageSelector
}

// Schema returns the OpenAPI schema for SoftwarePackageStrategy.
func (SoftwarePackageStrategy) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(softwarePackageStrategyValues...)
}

// Schema returns the OpenAPI schema for SoftwareAction.
func (SoftwareAction) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(softwareActionValues...)
}

func normalizeSoftwareTargets(targets SoftwareTargets) SoftwareTargets {
	if targets.Include == nil {
		targets.Include = []SoftwareInclude{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []targeting.LabelRef{}
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
	if err := include.Package.validate(); err != nil {
		return err
	}
	if len(include.Actions) == 0 {
		return fmt.Errorf("%w: actions is required", dbutil.ErrInvalidInput)
	}
	for _, action := range include.Actions {
		if !validSoftwareAction(action) {
			return fmt.Errorf("%w: unsupported action %q", dbutil.ErrInvalidInput, action)
		}
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

func validSoftwareAction(action SoftwareAction) bool {
	return slices.Contains(softwareActionValues, action)
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
