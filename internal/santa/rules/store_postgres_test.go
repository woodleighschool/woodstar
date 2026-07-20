//go:build postgres

package rules_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestRuleStorePersistsAndReplacesEditableShape(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)
	labelID := createSantaRuleLabel(t, db, "Santa Rule Validation")
	replacementLabelID := createSantaRuleLabel(t, db, "Santa Rule Replacement")
	excludeLabelID := createSantaRuleLabel(t, db, "Santa Rule Exclude")
	allHostsLabelID := santaRuleAllHostsLabelID(t, db)
	binaryIdentifier := strings.Repeat("a", 64)

	_, err := store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: strings.Repeat("b", 64),
		Name:       "Builtin Exclude",
		Targets: ruleTargets(
			[]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: labelID}},
			allHostsLabelID,
		),
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("create rule with builtin exclusion error = %v, want ErrInvalidInput", err)
	}

	rule, err := store.Create(ctx, rules.RuleMutation{
		RuleType:      rules.RuleTypeBinary,
		Identifier:    binaryIdentifier,
		Name:          "Example",
		Description:   "Example rule",
		CustomMessage: "Blocked",
		CustomURL:     "https://example.test",
		Targets: ruleTargets([]rules.RuleInclude{{
			Policy:  rules.PolicyAllowlist,
			LabelID: labelID,
		}}),
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if rule.Identifier != binaryIdentifier || rule.Name != "Example" || rule.Description != "Example rule" ||
		rule.CustomMessage != "Blocked" ||
		rule.CustomURL != "https://example.test" {
		t.Fatalf("rule = %+v, want persisted binary rule metadata", rule)
	}
	if len(rule.Targets.Include) != 1 || rule.Targets.Include[0].Policy != rules.PolicyAllowlist {
		t.Fatalf("include targets = %+v, want one allowlist include", rule.Targets.Include)
	}
	if rule.Targets.Exclude == nil {
		t.Fatalf("exclude targets = nil, want empty array")
	}

	_, err = store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: binaryIdentifier,
		Name:       "Duplicate",
	})
	if !errors.Is(err, dbutil.ErrAlreadyExists) {
		t.Fatalf("duplicate CreateRule error = %v, want ErrAlreadyExists", err)
	}

	celExpression := "target.path.startsWith('/Applications')"
	updated, err := store.Update(ctx, rule.ID, rules.RuleMutation{
		RuleType:      rules.RuleTypeSigningID,
		Identifier:    "ABCDE12345:com.example.updated",
		Name:          "Updated",
		Description:   "Updated rule",
		CustomMessage: "Updated message",
		Targets: ruleTargets([]rules.RuleInclude{{
			Policy:        rules.PolicyCEL,
			CELExpression: celExpression,
			LabelID:       replacementLabelID,
		}}, excludeLabelID),
	})
	if err != nil {
		t.Fatalf("update rule: %v", err)
	}
	if updated.RuleType != rules.RuleTypeSigningID || updated.Identifier != "ABCDE12345:com.example.updated" {
		t.Fatalf("update identity = %s %q, want signing id update", updated.RuleType, updated.Identifier)
	}
	if updated.Description != "Updated rule" {
		t.Fatalf("updated description = %q, want Updated rule", updated.Description)
	}
	if len(updated.Targets.Include) != 1 ||
		updated.Targets.Include[0].CELExpression != celExpression ||
		updated.Targets.Include[0].LabelID != replacementLabelID {
		t.Fatalf("updated include = %+v, want replacement label and CEL expression", updated.Targets.Include)
	}
	if len(updated.Targets.Exclude) != 1 || updated.Targets.Exclude[0].LabelID != excludeLabelID {
		t.Fatalf("exclude targets = %v, want [%d]", updated.Targets.Exclude, excludeLabelID)
	}
}

func TestRuleMissingLabelFallsThroughToNotFound(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)

	_, err := store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: strings.Repeat("d", 64),
		Name:       "Missing Include Label",
		Targets:    ruleTargets([]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: 999_999}}),
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("missing include label error = %v, want ErrNotFound", err)
	}

	labelID := createSantaRuleLabel(t, db, "Rule Missing Exclude Include")
	_, err = store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: strings.Repeat("e", 64),
		Name:       "Missing Exclude Label",
		Targets:    ruleTargets([]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: labelID}}, 999_999),
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("missing exclude label error = %v, want ErrNotFound", err)
	}
}

func TestRuleResolverUsesExcludeAndIncludePriority(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := rules.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-rule-resolver-host"},
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

	hostRule, err := store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Name:       "Targeted Binary",
		Identifier: strings.Repeat("1", 64),
		Targets: ruleTargets([]rules.RuleInclude{
			{Policy: rules.PolicyBlocklist, LabelID: firstLabelID},
			{Policy: rules.PolicySilentBlocklist, LabelID: secondLabelID},
		}),
	})
	if err != nil {
		t.Fatalf("create host rule: %v", err)
	}
	excludedRule, err := store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeTeamID,
		Identifier: "TEAMID1234",
		Name:       "Excluded Team",
		Targets: ruleTargets(
			[]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: secondLabelID}},
			excludeLabelID,
		),
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
		t.Fatalf("host rules = %+v, want exactly one", got)
	}
	if got[0].RuleID != hostRule.ID || got[0].Name != "Targeted Binary" ||
		got[0].Policy != rules.PolicySilentBlocklist {
		t.Fatalf("host rule = %+v, want second include to win", got[0])
	}
	if got[0].RuleID == excludedRule.ID {
		t.Fatalf("excluded rule resolved: %+v", got[0])
	}
}

func TestRuleResolverAllowsAllHostsInclude(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := rules.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-rule-all-hosts"},
		OrbitNodeKey: "santa-rule-all-hosts-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	allHostsLabelID := santaRuleAllHostsLabelID(t, db)

	rule, err := store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeTeamID,
		Identifier: "ALLHOST123",
		Name:       "All Hosts Team",
		Targets:    ruleTargets([]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: allHostsLabelID}}),
	})
	if err != nil {
		t.Fatalf("create all hosts rule: %v", err)
	}

	got, err := store.ResolveRulesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve rules: %v", err)
	}
	if len(got) != 1 || got[0].RuleID != rule.ID {
		t.Fatalf("host rules = %+v, want all hosts rule", got)
	}
}

func TestListRuleStatusesForHostMissingHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)

	_, _, err := store.ListRuleStatusesForHost(ctx, 999999, rules.RuleStatusListParams{})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("ListRuleStatusesForHost missing host error = %v, want ErrNotFound", err)
	}
}

func TestListRuleStatusesForHostEmptyHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := rules.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-rule-status-empty-host"},
		OrbitNodeKey: "santa-rule-status-empty-host-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	rows, count, err := store.ListRuleStatusesForHost(ctx, host.ID, rules.RuleStatusListParams{})
	if err != nil {
		t.Fatalf("ListRuleStatusesForHost empty host: %v", err)
	}
	if len(rows) != 0 || count != 0 {
		t.Fatalf("ListRuleStatusesForHost empty host = %d rows count %d, want empty page", len(rows), count)
	}
}

func TestBundleRuleExpandsToBinaryHostRules(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := rules.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-bundle-rule-host"},
		OrbitNodeKey: "santa-bundle-rule-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	labelID := createSantaRuleLabel(t, db, "Santa Bundle Rule")
	if err := labelStore.SetMembership(ctx, labelID, host.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}

	bundleHash := strings.Repeat("b", 64)
	firstSHA := strings.Repeat("1", 64)
	secondSHA := strings.Repeat("2", 64)
	var firstExecutableID int64
	var secondExecutableID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_executables (sha256, file_name)
		VALUES ($1, 'Bundle Main')
		RETURNING id
	`, firstSHA).Scan(&firstExecutableID); err != nil {
		t.Fatalf("insert first executable: %v", err)
	}
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_executables (sha256, file_name)
		VALUES ($1, 'Bundle Helper')
		RETURNING id
	`, secondSHA).Scan(&secondExecutableID); err != nil {
		t.Fatalf("insert second executable: %v", err)
	}
	var bundleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_bundles (
			sha256,
			bundle_id,
			name,
			path,
			version,
			version_string,
			binary_count,
			uploaded_at
		)
		VALUES ($1, 'com.example.bundle-rule', 'Bundle Rule App', '/Applications/Bundle Rule.app', '4.5.6', '4.5.6 (7)', 2, now())
		RETURNING id
	`, bundleHash).Scan(&bundleID); err != nil {
		t.Fatalf("insert bundle: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_bundle_executables (bundle_id, executable_id)
		VALUES ($1, $2), ($1, $3)
	`, bundleID, firstExecutableID, secondExecutableID); err != nil {
		t.Fatalf("link bundle executables: %v", err)
	}

	rule, err := store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBundle,
		Identifier: bundleHash,
		Name:       "Bundle Rule",
		Targets:    ruleTargets([]rules.RuleInclude{{Policy: rules.PolicyBlocklist, LabelID: labelID}}),
	})
	if err != nil {
		t.Fatalf("create bundle rule: %v", err)
	}

	got, err := store.ResolveRulesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve rules: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("host rules = %+v, want two binary expansions", got)
	}
	for _, hostRule := range got {
		if hostRule.RuleID != rule.ID ||
			hostRule.RuleType != rules.RuleTypeBinary ||
			hostRule.Policy != rules.PolicyBlocklist ||
			hostRule.AppName != "Bundle Rule App" {
			t.Fatalf("expanded rule = %+v", hostRule)
		}
	}
	if got[0].Identifier != firstSHA || got[1].Identifier != secondSHA {
		t.Fatalf("expanded identifiers = %q/%q, want bundle executables", got[0].Identifier, got[1].Identifier)
	}

	targets := rules.SyncTargetsFromRules(got)
	if len(targets) != 2 || targets[0].RuleType != string(rules.RuleTypeBinary) ||
		targets[0].AppName != "Bundle Rule App" {
		t.Fatalf("sync targets = %+v, want binary payloads carrying bundle notification data", targets)
	}

	if _, err := db.Pool().Exec(ctx, `UPDATE santa_bundles SET name = '' WHERE id = $1`, bundleID); err != nil {
		t.Fatalf("clear bundle name: %v", err)
	}
	got, err = store.ResolveRulesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve unnamed bundle rule: %v", err)
	}
	for _, hostRule := range got {
		if hostRule.AppName != "" {
			t.Fatalf("unnamed bundle notification app name = %q, want empty", hostRule.AppName)
		}
	}
}

func TestRuleStoreBulkDeleteIgnoresMissingIDs(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)

	first, err := store.Create(
		ctx,
		rules.RuleMutation{RuleType: rules.RuleTypeBinary, Identifier: strings.Repeat("3", 64), Name: "Bulk Binary"},
	)
	if err != nil {
		t.Fatalf("create first rule: %v", err)
	}
	second, err := store.Create(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeTeamID,
		Identifier: "BULKTEAM12",
		Name:       "Bulk Team",
	})
	if err != nil {
		t.Fatalf("create second rule: %v", err)
	}

	deleted, err := store.DeleteMany(ctx, []int64{first.ID, second.ID + 999})
	if err != nil {
		t.Fatalf("bulk delete: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := store.GetByID(ctx, first.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("deleted rule lookup error = %v, want ErrNotFound", err)
	}
	if _, err := store.GetByID(ctx, second.ID); err != nil {
		t.Fatalf("kept rule lookup: %v", err)
	}
}

func createSantaRuleLabel(t *testing.T, db *database.DB, name string) int64 {
	t.Helper()

	label, err := labels.NewStore(db).Create(t.Context(), labels.LabelMutation{
		Name:                name,
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label %q: %v", name, err)
	}
	return label.ID
}

func santaRuleAllHostsLabelID(t *testing.T, db *database.DB) int64 {
	t.Helper()

	var id int64
	err := db.Pool().QueryRow(
		t.Context(),
		`SELECT id FROM labels WHERE builtin_key = $1 AND label_type = 'builtin'`,
		string(labels.BuiltinKeyAllHosts),
	).Scan(&id)
	if err != nil {
		t.Fatalf("get All Hosts label: %v", err)
	}
	return id
}
