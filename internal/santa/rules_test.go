package santa_test

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/santa"
)

func TestRuleStoreValidatesAndReplacesEditableShape(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := santa.NewStore(db)
	labelID := createSantaRuleLabel(t, db, "Santa Rule Validation")

	invalidCases := []struct {
		name   string
		params santa.RuleCreate
	}{
		{name: "missing type", params: santa.RuleCreate{Identifier: "abc123"}},
		{name: "missing identifier", params: santa.RuleCreate{RuleType: santa.RuleTypeBinary}},
		{
			name: "cel without expression",
			params: santa.RuleCreate{
				RuleType:   santa.RuleTypeBinary,
				Identifier: "abc123",
				Includes:   []santa.RuleIncludeWrite{{Policy: santa.PolicyCEL, LabelIDs: []int64{labelID}}},
			},
		},
		{
			name: "non cel with expression",
			params: santa.RuleCreate{
				RuleType:   santa.RuleTypeBinary,
				Identifier: "abc123",
				Includes: []santa.RuleIncludeWrite{{
					Policy:        santa.PolicyAllowlist,
					CELExpression: "target.path == '/Applications'",
					LabelIDs:      []int64{labelID},
				}},
			},
		},
		{
			name: "include without labels",
			params: santa.RuleCreate{
				RuleType:   santa.RuleTypeBinary,
				Identifier: "abc123",
				Includes:   []santa.RuleIncludeWrite{{Policy: santa.PolicyAllowlist}},
			},
		},
	}
	for _, tt := range invalidCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.CreateRule(ctx, tt.params)
			if !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("CreateRule error = %v, want ErrInvalidInput", err)
			}
		})
	}

	rule, err := store.CreateRule(ctx, santa.RuleCreate{
		RuleType:      santa.RuleTypeBinary,
		Identifier:    " abc123 ",
		Name:          " Example ",
		CustomMessage: " Blocked ",
		CustomURL:     " https://example.test ",
		Includes: []santa.RuleIncludeWrite{{
			Policy:   santa.PolicyAllowlist,
			LabelIDs: []int64{labelID},
		}},
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if rule.Identifier != "abc123" || rule.Name != "Example" || rule.CustomMessage != "Blocked" ||
		rule.CustomURL != "https://example.test" {
		t.Fatalf("rule was not cleaned: %+v", rule)
	}
	if len(rule.Includes) != 1 || rule.Includes[0].Position != 0 || rule.Includes[0].Policy != santa.PolicyAllowlist {
		t.Fatalf("includes = %+v, want one allowlist include at position 0", rule.Includes)
	}

	_, err = store.CreateRule(ctx, santa.RuleCreate{
		RuleType:   santa.RuleTypeBinary,
		Identifier: "abc123",
	})
	if !errors.Is(err, dbutil.ErrAlreadyExists) {
		t.Fatalf("duplicate CreateRule error = %v, want ErrAlreadyExists", err)
	}

	celExpression := "target.path.startsWith('/Applications')"
	updated, err := store.UpdateRule(ctx, rule.ID, santa.RuleUpdate{
		Name:          "Updated",
		CustomMessage: "Updated message",
		Includes: []santa.RuleIncludeWrite{{
			Policy:        santa.PolicyCEL,
			CELExpression: celExpression,
			LabelIDs:      []int64{labelID},
		}},
		ExcludeLabelIDs: []int64{labelID},
	})
	if err != nil {
		t.Fatalf("update rule: %v", err)
	}
	if updated.RuleType != santa.RuleTypeBinary || updated.Identifier != "abc123" {
		t.Fatalf("update changed immutable identity: %+v", updated)
	}
	if len(updated.Includes) != 1 || updated.Includes[0].CELExpression != celExpression {
		t.Fatalf("updated include = %+v, want CEL expression", updated.Includes)
	}
	if len(updated.ExcludeLabelIDs) != 1 || updated.ExcludeLabelIDs[0] != labelID {
		t.Fatalf("exclude labels = %v, want [%d]", updated.ExcludeLabelIDs, labelID)
	}
}

func TestRuleResolverUsesExcludeAndIncludePriority(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := santa.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-rule-resolver-host",
		OrbitNodeKey: "santa-rule-resolver-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	firstLabelID := createSantaRuleLabel(t, db, "Santa Resolver First")
	secondLabelID := createSantaRuleLabel(t, db, "Santa Resolver Second")
	excludeLabelID := createSantaRuleLabel(t, db, "Santa Resolver Exclude")
	if err := labelStore.SetMembership(ctx, secondLabelID, host.ID, true); err != nil {
		t.Fatalf("set second label membership: %v", err)
	}

	effectiveRule, err := store.CreateRule(ctx, santa.RuleCreate{
		RuleType:   santa.RuleTypeBinary,
		Identifier: "binary-sha",
		Includes: []santa.RuleIncludeWrite{
			{Policy: santa.PolicyBlocklist, LabelIDs: []int64{firstLabelID}},
			{Policy: santa.PolicySilentBlocklist, LabelIDs: []int64{secondLabelID}},
		},
	})
	if err != nil {
		t.Fatalf("create effective rule: %v", err)
	}
	excludedRule, err := store.CreateRule(ctx, santa.RuleCreate{
		RuleType:        santa.RuleTypeTeamID,
		Identifier:      "TEAMID",
		Includes:        []santa.RuleIncludeWrite{{Policy: santa.PolicyAllowlist, LabelIDs: []int64{secondLabelID}}},
		ExcludeLabelIDs: []int64{excludeLabelID},
	})
	if err != nil {
		t.Fatalf("create excluded rule: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excludeLabelID, host.ID, true); err != nil {
		t.Fatalf("set exclude label membership: %v", err)
	}

	got, err := store.ResolveRulesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve rules: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("effective rules = %+v, want exactly one", got)
	}
	if got[0].RuleID != effectiveRule.ID || got[0].Policy != santa.PolicySilentBlocklist {
		t.Fatalf("effective rule = %+v, want second include to win", got[0])
	}
	if got[0].RuleID == excludedRule.ID {
		t.Fatalf("excluded rule resolved: %+v", got[0])
	}
}

func TestRuleIncludeReorderRequiresExactSet(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := santa.NewStore(db)
	firstLabelID := createSantaRuleLabel(t, db, "Santa Reorder First")
	secondLabelID := createSantaRuleLabel(t, db, "Santa Reorder Second")

	rule, err := store.CreateRule(ctx, santa.RuleCreate{
		RuleType:   santa.RuleTypeCertificate,
		Identifier: "certificate-sha",
		Includes: []santa.RuleIncludeWrite{
			{Policy: santa.PolicyAllowlist, LabelIDs: []int64{firstLabelID}},
			{Policy: santa.PolicyBlocklist, LabelIDs: []int64{secondLabelID}},
		},
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	err = store.ReorderRuleIncludes(ctx, rule.ID, []int64{rule.Includes[0].ID})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("partial reorder error = %v, want ErrInvalidInput", err)
	}
	if err := store.ReorderRuleIncludes(ctx, rule.ID+9999, nil); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("missing rule reorder error = %v, want ErrNotFound", err)
	}
	if err := store.ReorderRuleIncludes(ctx, rule.ID, []int64{rule.Includes[1].ID, rule.Includes[0].ID}); err != nil {
		t.Fatalf("reorder includes: %v", err)
	}

	got, err := store.GetRuleByID(ctx, rule.ID)
	if err != nil {
		t.Fatalf("get rule: %v", err)
	}
	if got.Includes[0].ID != rule.Includes[1].ID || got.Includes[0].Position != 0 {
		t.Fatalf("includes after reorder = %+v", got.Includes)
	}
}

func createSantaRuleLabel(t *testing.T, db *database.DB, name string) int64 {
	t.Helper()

	label, err := labels.NewStore(db).Create(t.Context(), labels.LabelCreate{
		Name:                name,
		LabelType:           labels.LabelTypeRegular,
		LabelMembershipType: labels.LabelMembershipTypeManual,
		Platforms: []platforms.Platform{
			platforms.PlatformDarwin,
			platforms.PlatformWindows,
			platforms.PlatformLinux,
		},
	})
	if err != nil {
		t.Fatalf("create label %q: %v", name, err)
	}
	return label.ID
}
