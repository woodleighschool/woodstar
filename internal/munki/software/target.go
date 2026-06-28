package software

import (
	"fmt"
	"slices"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/schema"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// Targets is the include/exclude label targeting contract for Munki software.
type Targets struct {
	Include []Include            `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

// Include applies desired Munki manifest actions to hosts with a matching label.
type Include struct {
	LabelID int64           `json:"label_id" minimum:"1"`
	Package PackageSelector `json:"package"`
	Actions []Action        `json:"actions"              minItems:"1" nullable:"false"`
}

// PackageSelector chooses the package candidate set for a software include.
type PackageSelector struct {
	Strategy  PackageStrategy `json:"strategy"`
	PackageID *int64          `json:"package_id,omitempty" minimum:"1"`
}

// PackageStrategy describes whether Munki software follows the latest
// eligible package or pins one package version.
type PackageStrategy string

const (
	PackageLatest   PackageStrategy = "latest"
	PackageSpecific PackageStrategy = "specific"
)

var packageStrategyValues = []PackageStrategy{
	PackageLatest,
	PackageSpecific,
}

// Action describes one Munki manifest section for an include.
type Action string

const (
	ActionManagedInstalls   Action = "managed_installs"
	ActionManagedUninstalls Action = "managed_uninstalls"
	ActionManagedUpdates    Action = "managed_updates"
	ActionOptionalInstalls  Action = "optional_installs"
	ActionFeaturedItems     Action = "featured_items"
	ActionDefaultInstalls   Action = "default_installs"
)

var actionValues = []Action{
	ActionManagedInstalls,
	ActionManagedUninstalls,
	ActionManagedUpdates,
	ActionOptionalInstalls,
	ActionFeaturedItems,
	ActionDefaultInstalls,
}

// EffectivePackage is a host-resolved Munki package ready for manifest/catalog rendering.
type EffectivePackage struct {
	TargetID             int64
	SoftwareID           int64
	Actions              []Action
	Package              packages.Package
	SoftwareIconObjectID *int64
	Selector             PackageSelector
}

// Schema returns the OpenAPI schema for PackageStrategy.
func (PackageStrategy) Schema(_ huma.Registry) *huma.Schema {
	return schema.StringEnum(packageStrategyValues...)
}

// Schema returns the OpenAPI schema for Action.
func (Action) Schema(_ huma.Registry) *huma.Schema {
	return schema.StringEnum(actionValues...)
}

func normalizeTargets(targets Targets) Targets {
	if targets.Include == nil {
		targets.Include = []Include{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []targeting.LabelRef{}
	}
	return targets
}

func emptyTargets() Targets {
	return Targets{
		Include: []Include{},
		Exclude: []targeting.LabelRef{},
	}
}

func (targets Targets) validate() error {
	for _, include := range targets.Include {
		if err := include.validate(); err != nil {
			return err
		}
	}
	if err := targeting.ValidateTargets(targets.Include, targets.Exclude, includeLabelID); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (include Include) validate() error {
	if !validPackageStrategy(include.Package.Strategy) {
		return fmt.Errorf("%w: package.strategy is required", dbutil.ErrInvalidInput)
	}
	if err := include.Package.validate(); err != nil {
		return err
	}
	if len(include.Actions) == 0 {
		return fmt.Errorf("%w: actions is required", dbutil.ErrInvalidInput)
	}
	for _, action := range include.Actions {
		if !validAction(action) {
			return fmt.Errorf("%w: unsupported action %q", dbutil.ErrInvalidInput, action)
		}
	}
	return nil
}

func (selector PackageSelector) validate() error {
	switch selector.Strategy {
	case PackageLatest:
		if selector.PackageID != nil {
			return fmt.Errorf("%w: package.package_id must be empty for latest strategy", dbutil.ErrInvalidInput)
		}
	case PackageSpecific:
		if selector.PackageID == nil {
			return fmt.Errorf("%w: package.package_id is required for specific strategy", dbutil.ErrInvalidInput)
		}
		if *selector.PackageID <= 0 {
			return fmt.Errorf("%w: package.package_id must be positive", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

func validPackageStrategy(strategy PackageStrategy) bool {
	return slices.Contains(packageStrategyValues, strategy)
}

func validAction(action Action) bool {
	return slices.Contains(actionValues, action)
}

func includeLabelID(include Include) int64 {
	return include.LabelID
}
