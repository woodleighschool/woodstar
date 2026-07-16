package rules

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/database/dbtest/crudtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestRuleStoreConformance(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	crudtest.RunConformance(
		t,
		ctx,
		crudtest.Fixtures[Rule, RuleMutation, RuleMutation, RuleListParams]{
			Store: store,
			NewValid: func(t *testing.T, ctx context.Context) RuleMutation {
				t.Helper()

				label, err := labels.NewStore(db).Create(ctx, labels.LabelMutation{
					Name:                "Conformance Rule Label",
					LabelMembershipType: labels.LabelMembershipTypeManual,
				})
				if err != nil {
					t.Fatalf("create label: %v", err)
				}
				return RuleMutation{
					RuleType:   RuleTypeBinary,
					Identifier: strings.Repeat("c", 64),
					Name:       "Conformance Rule",
					Targets: RuleTargets{
						Include: []RuleInclude{{Policy: PolicyAllowlist, LabelID: label.ID}},
						Exclude: []targeting.LabelRef{},
					},
				}
			},
			Mutate: func(r Rule) RuleMutation {
				return RuleMutation{
					RuleType:    RuleTypeTeamID,
					Identifier:  "ABCDE12345",
					Name:        "Conformance Rule Updated",
					Description: "updated",
					Targets:     r.Targets,
				}
			},
			ID:         func(r Rule) int64 { return r.ID },
			ListParams: ruleListParams,
			SortKeys:   slices.Sorted(maps.Keys(ruleOrderKeys())),
			SearchMatch: func(r Rule) string {
				return r.Name
			},
			NewInvalid: func() (RuleMutation, bool) {
				return RuleMutation{RuleType: RuleType("bogus"), Identifier: strings.Repeat("d", 64), Name: "Bad"}, true
			},
		},
	)
}

func ruleListParams(q, sort string, pageIndex, pageSize int32) RuleListParams {
	return RuleListParams{
		ListParams: dbutil.ListParams{
			Q:         q,
			Sort:      sort,
			PageIndex: pageIndex,
			PageSize:  pageSize,
		},
	}
}
