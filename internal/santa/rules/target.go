package rules

import (
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// RuleTargets is the include/exclude label targeting contract for a Santa rule.
type RuleTargets struct {
	Include []RuleInclude        `json:"include" nullable:"false" validate:"dive"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

// RuleInclude applies a rule policy to hosts with a matching label.
type RuleInclude struct {
	Policy        Policy `json:"policy"                   validate:"required,oneof=allowlist allowlist_compiler blocklist silent_blocklist silent_gui_blocklist silent_tty_blocklist cel"`
	CELExpression string `json:"cel_expression,omitempty" validate:"excluded_unless=Policy cel,required_if=Policy cel"`
	LabelID       int64  `json:"label_id"                 validate:"gt=0"                                                                                                                 minimum:"1"`
}

func (targets RuleTargets) validate() error {
	if err := validation.Struct(targets); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	for _, include := range targets.Include {
		if include.Policy == PolicyCEL {
			if err := validateCELSyntax(include.CELExpression); err != nil {
				return err
			}
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
	for i := range targets.Include {
		targets.Include[i].CELExpression = strings.TrimSpace(targets.Include[i].CELExpression)
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
