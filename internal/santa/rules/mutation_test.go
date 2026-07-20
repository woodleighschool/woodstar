package rules_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestRuleMutationValidatesCELSyntax(t *testing.T) {
	base := rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: strings.Repeat("a", 64),
		Name:       "CEL Rule",
		Targets: ruleTargets([]rules.RuleInclude{{
			Policy:        rules.PolicyCEL,
			CELExpression: "target.signing_id == 'ABCDE12345:com.example.app'",
			LabelID:       1,
		}}),
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("valid CEL expression error = %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*rules.RuleMutation)
	}{
		{
			name: "malformed cel",
			mutate: func(params *rules.RuleMutation) {
				params.Targets.Include[0].CELExpression = "target.signing_id =="
			},
		},
		{
			name: "empty cel",
			mutate: func(params *rules.RuleMutation) {
				params.Targets.Include[0].CELExpression = ""
			},
		},
		{
			name: "blank cel",
			mutate: func(params *rules.RuleMutation) {
				params.Targets.Include[0].CELExpression = "  "
			},
		},
		{
			name: "non cel with expression",
			mutate: func(params *rules.RuleMutation) {
				params.Targets.Include[0].Policy = rules.PolicyAllowlist
			},
		},
		{
			name: "bad signing id identifier",
			mutate: func(params *rules.RuleMutation) {
				params.RuleType = rules.RuleTypeSigningID
				params.Identifier = "not-a-signing-id"
			},
		},
		{
			name: "bad cdhash identifier",
			mutate: func(params *rules.RuleMutation) {
				params.RuleType = rules.RuleTypeCDHash
				params.Identifier = strings.Repeat("a", 39)
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			params := base
			params.Targets.Include = append([]rules.RuleInclude(nil), base.Targets.Include...)
			params.Targets.Exclude = append([]targeting.LabelRef(nil), base.Targets.Exclude...)
			tt.mutate(&params)

			err := params.Validate()
			if !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("Validate error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestRuleMutationValidatesShapeAndTargets(t *testing.T) {
	t.Parallel()

	base := rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: strings.Repeat("a", 64),
		Name:       "Example",
		Targets: ruleTargets([]rules.RuleInclude{{
			Policy:  rules.PolicyAllowlist,
			LabelID: 1,
		}}),
	}
	cases := []struct {
		name   string
		mutate func(*rules.RuleMutation)
	}{
		{name: "missing type", mutate: func(m *rules.RuleMutation) { m.RuleType = "" }},
		{name: "missing identifier", mutate: func(m *rules.RuleMutation) { m.Identifier = "" }},
		{name: "missing name", mutate: func(m *rules.RuleMutation) { m.Name = "" }},
		{name: "duplicate include label", mutate: func(m *rules.RuleMutation) {
			m.Targets.Include = append(m.Targets.Include, rules.RuleInclude{Policy: rules.PolicyBlocklist, LabelID: 1})
		}},
		{name: "duplicate exclude label", mutate: func(m *rules.RuleMutation) {
			m.Targets.Exclude = ruleLabelRefs(2, 2)
		}},
		{name: "include and exclude overlap", mutate: func(m *rules.RuleMutation) {
			m.Targets.Exclude = ruleLabelRefs(1)
		}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := base
			mutation.Targets.Include = append([]rules.RuleInclude(nil), base.Targets.Include...)
			tt.mutate(&mutation)
			if err := mutation.Validate(); !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("Validate error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func ruleTargets(includes []rules.RuleInclude, excludedLabelIDs ...int64) rules.RuleTargets {
	return rules.RuleTargets{
		Include: includes,
		Exclude: ruleLabelRefs(excludedLabelIDs...),
	}
}

func ruleLabelRefs(labelIDs ...int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(labelIDs))
	for i, labelID := range labelIDs {
		refs[i] = targeting.LabelRef{LabelID: labelID}
	}
	return refs
}
