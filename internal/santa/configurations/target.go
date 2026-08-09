package configurations

import (
	"fmt"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// ConfigurationTargets is the include/exclude label targeting contract for a Santa configuration.
type ConfigurationTargets struct {
	Include []targeting.LabelRef `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

func (targets ConfigurationTargets) validate() error {
	if err := targeting.ValidateLabelSets(targets.Include, targets.Exclude); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

func normalizeConfigurationTargets(targets ConfigurationTargets) ConfigurationTargets {
	if targets.Include == nil {
		targets.Include = []targeting.LabelRef{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []targeting.LabelRef{}
	}
	return targets
}

func emptyConfigurationTargets() ConfigurationTargets {
	return ConfigurationTargets{
		Include: []targeting.LabelRef{},
		Exclude: []targeting.LabelRef{},
	}
}
