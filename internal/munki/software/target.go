package software

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// Targets is the include/exclude label targeting contract for Munki software.
type Targets struct {
	Include []Include            `json:"include" nullable:"false" validate:"dive"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

// Include applies desired Munki manifest actions to hosts with a matching label.
type Include struct {
	LabelID int64           `json:"label_id" minimum:"1" validate:"gt=0"`
	Package PackageSelector `json:"package"`
	Actions []Action        `json:"actions"              validate:"min=1,dive,oneof=managed_installs managed_uninstalls managed_updates optional_installs featured_items default_installs" minItems:"1" nullable:"false"`
}

// PackageSelector chooses the package candidate set for a software include.
type PackageSelector struct {
	Strategy  PackageStrategy `json:"strategy"             validate:"required,oneof=latest specific"`
	PackageID *int64          `json:"package_id,omitempty" validate:"excluded_unless=Strategy specific,required_if=Strategy specific,omitempty,gt=0" minimum:"1"`
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
	return openapischema.StringEnum(packageStrategyValues...)
}

// Schema returns the OpenAPI schema for Action.
func (Action) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(actionValues...)
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
	if err := validation.Struct(targets); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	if err := targeting.ValidateTargets(targets.Include, targets.Exclude, includeLabelID); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func includeLabelID(include Include) int64 {
	return include.LabelID
}
