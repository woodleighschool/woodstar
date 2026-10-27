package rules

import (
	"fmt"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// RuleTargets is the include/exclude label targeting contract for a Santa rule.
type RuleTargets struct {
	Include []targeting.LabelRef `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

func (targets RuleTargets) validate() error {
	if err := targeting.ValidateLabelSets(targets.Include, targets.Exclude); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

func normalizeRuleTargets(targets RuleTargets) RuleTargets {
	if targets.Include == nil {
		targets.Include = []targeting.LabelRef{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []targeting.LabelRef{}
	}
	return targets
}

func emptyRuleTargets() RuleTargets {
	return RuleTargets{
		Include: []targeting.LabelRef{},
		Exclude: []targeting.LabelRef{},
	}
}
