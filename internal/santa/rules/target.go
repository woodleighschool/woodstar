package rules

import (
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// RuleTargets is the include/exclude label targeting contract for a Santa rule.
type RuleTargets struct {
	Include []RuleInclude        `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

// RuleInclude applies a rule policy to hosts with a matching label.
type RuleInclude struct {
	Policy        Policy `json:"policy"`
	CELExpression string `json:"cel_expression,omitempty"`
	LabelID       int64  `json:"label_id"                 minimum:"1"`
}

func (targets RuleTargets) validate() error {
	for _, include := range targets.Include {
		if !validPolicy(include.Policy) {
			return fmt.Errorf("%w: include policy is required", dbutil.ErrInvalidInput)
		}
		if include.Policy == PolicyCEL && strings.TrimSpace(include.CELExpression) == "" {
			return fmt.Errorf("%w: cel_expression is required for cel policy", dbutil.ErrInvalidInput)
		}
		if include.Policy == PolicyCEL {
			if err := validateCELSyntax(include.CELExpression); err != nil {
				return err
			}
		}
		if include.Policy != PolicyCEL && include.CELExpression != "" {
			return fmt.Errorf("%w: cel_expression is only valid for cel policy", dbutil.ErrInvalidInput)
		}
	}
	if err := targeting.ValidateTargets(targets.Include, targets.Exclude, ruleIncludeLabelID); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func normalizeRuleTargets(targets RuleTargets) RuleTargets {
	if targets.Include == nil {
		targets.Include = []RuleInclude{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []targeting.LabelRef{}
	}
	return targets
}

func emptyRuleTargets() RuleTargets {
	return RuleTargets{
		Include: []RuleInclude{},
		Exclude: []targeting.LabelRef{},
	}
}

func ruleIncludeLabelID(include RuleInclude) int64 {
	return include.LabelID
}

func labelRefIDs(refs []targeting.LabelRef) []int64 {
	ids := make([]int64, len(refs))
	for i, ref := range refs {
		ids[i] = ref.LabelID
	}
	return ids
}
