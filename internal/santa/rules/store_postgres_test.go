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
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestRuleStorePersistsAndReplacesEditableShape(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)
	configurationID := createSantaRuleConfiguration(t, db, "Santa Rule Validation")
	labelID := createSantaRuleLabel(t, db, "Santa Rule Validation")
	replacementLabelID := createSantaRuleLabel(t, db, "Santa Rule Replacement")
	excludeLabelID := createSantaRuleLabel(t, db, "Santa Rule Exclude")
	allHostsLabelID := santaRuleAllHostsLabelID(t, db)
	binaryIdentifier := strings.Repeat("a", 64)

	_, err := store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeBinary,
		Identifier:      strings.Repeat("b", 64),
		Name:            "Builtin Exclude",
		Policy:          rules.PolicyAllowlist,
		Targets:         storeRuleTargets([]int64{labelID}, allHostsLabelID),
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("create rule with builtin exclusion error = %v, want ErrInvalidInput", err)
	}

	rule, err := store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeBinary,
		Identifier:      binaryIdentifier,
		Name:            "Example",
		Description:     "Example rule",
		Policy:          rules.PolicyAllowlist,
		CustomMessage:   "Blocked",
		CustomURL:       "https://example.test",
		Targets:         storeRuleTargets([]int64{labelID}),
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if rule.ConfigurationID != configurationID || rule.Identifier != binaryIdentifier ||
		rule.Name != "Example" || rule.Description != "Example rule" ||
		rule.Policy != rules.PolicyAllowlist ||
		rule.CustomMessage != "Blocked" ||
		rule.CustomURL != "https://example.test" {
		t.Fatalf("rule = %+v, want persisted binary rule metadata", rule)
	}
	if len(rule.Targets.Include) != 1 || rule.Targets.Include[0].LabelID != labelID {
		t.Fatalf("include targets = %+v, want label %d", rule.Targets.Include, labelID)
	}
	if rule.Targets.Exclude == nil {
		t.Fatalf("exclude targets = nil, want empty array")
	}

	_, err = store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeBinary,
		Identifier:      binaryIdentifier,
		Name:            "Duplicate",
		Policy:          rules.PolicyBlocklist,
	})
	if !errors.Is(err, dbutil.ErrAlreadyExists) {
		t.Fatalf("duplicate CreateRule error = %v, want ErrAlreadyExists", err)
	}

	celExpression := "target.path.startsWith('/Applications')"
	updated, err := store.Update(ctx, rule.ID, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeSigningID,
		Identifier:      "ABCDE12345:com.example.updated",
		Name:            "Updated",
		Description:     "Updated rule",
		Policy:          rules.PolicyCEL,
		CELExpression:   celExpression,
		CustomMessage:   "Updated message",
		Targets:         storeRuleTargets([]int64{replacementLabelID}, excludeLabelID),
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
	if updated.Policy != rules.PolicyCEL || updated.CELExpression != celExpression ||
		len(updated.Targets.Include) != 1 ||
		updated.Targets.Include[0].LabelID != replacementLabelID {
		t.Fatalf("updated rule = %+v, want complete CEL rule with replacement label", updated)
	}
	if len(updated.Targets.Exclude) != 1 || updated.Targets.Exclude[0].LabelID != excludeLabelID {
		t.Fatalf("exclude targets = %v, want [%d]", updated.Targets.Exclude, excludeLabelID)
	}
}

func TestRuleMissingLabelFallsThroughToNotFound(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)
	configurationID := createSantaRuleConfiguration(t, db, "Santa Rule Missing Label")

	_, err := store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeBinary,
		Identifier:      strings.Repeat("d", 64),
		Name:            "Missing Include Label",
		Policy:          rules.PolicyAllowlist,
		Targets:         storeRuleTargets([]int64{999_999}),
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("missing include label error = %v, want ErrNotFound", err)
	}

	labelID := createSantaRuleLabel(t, db, "Rule Missing Exclude Include")
	_, err = store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeBinary,
		Identifier:      strings.Repeat("e", 64),
		Name:            "Missing Exclude Label",
		Policy:          rules.PolicyAllowlist,
		Targets:         storeRuleTargets([]int64{labelID}, 999_999),
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("missing exclude label error = %v, want ErrNotFound", err)
	}
}

func TestRuleResolverUsesCompleteDecisionAcrossIncludesAndExcludes(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := rules.NewStore(db)
	configurationID := createSantaRuleConfiguration(t, db, "Santa Rule Resolver")

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
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeBinary,
		Name:            "Targeted Binary",
		Identifier:      strings.Repeat("1", 64),
		Policy:          rules.PolicySilentBlocklist,
		Targets:         storeRuleTargets([]int64{firstLabelID, secondLabelID}),
	})
	if err != nil {
		t.Fatalf("create host rule: %v", err)
	}
	excludedRule, err := store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeTeamID,
		Identifier:      "TEAMID1234",
		Name:            "Excluded Team",
		Policy:          rules.PolicyAllowlist,
		Targets:         storeRuleTargets([]int64{secondLabelID}, excludeLabelID),
	})
	if err != nil {
		t.Fatalf("create excluded rule: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excludeLabelID, host.ID, true); err != nil {
		t.Fatalf("set exclude label membership: %v", err)
	}

	got, err := store.ResolveRulesForHost(ctx, host.ID, configurationID)
	if err != nil {
		t.Fatalf("resolve rules: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("host rules = %+v, want exactly one", got)
	}
	if got[0].RuleID != hostRule.ID || got[0].Name != "Targeted Binary" ||
		got[0].Policy != rules.PolicySilentBlocklist {
		t.Fatalf("host rule = %+v, want the rule-owned decision", got[0])
	}
	if got[0].RuleID == excludedRule.ID {
		t.Fatalf("excluded rule resolved: %+v", got[0])
	}
}

func TestRuleResolverAllowsAllHostsInclude(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := rules.NewStore(db)
	configurationID := createSantaRuleConfiguration(t, db, "Santa All Hosts Rule")

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-rule-all-hosts"},
		OrbitNodeKey: "santa-rule-all-hosts-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	allHostsLabelID := santaRuleAllHostsLabelID(t, db)

	rule, err := store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeTeamID,
		Identifier:      "ALLHOST123",
		Name:            "All Hosts Team",
		Policy:          rules.PolicyAllowlist,
		Targets:         storeRuleTargets([]int64{allHostsLabelID}),
	})
	if err != nil {
		t.Fatalf("create all hosts rule: %v", err)
	}

	got, err := store.ResolveRulesForHost(ctx, host.ID, configurationID)
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

	_, _, err := store.ListRuleStatusesForHost(ctx, 999999, 0, rules.RuleStatusListParams{})
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

	rows, count, err := store.ListRuleStatusesForHost(ctx, host.ID, 0, rules.RuleStatusListParams{})
	if err != nil {
		t.Fatalf("ListRuleStatusesForHost empty host: %v", err)
	}
	if len(rows) != 0 || count != 0 {
		t.Fatalf("ListRuleStatusesForHost empty host = %d rows count %d, want empty page", len(rows), count)
	}
}

func TestRuleStatusesCanonicalizeBundleCollisions(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	ruleStore := rules.NewStore(db)
	configurationID := createSantaRuleConfiguration(t, db, "Santa Rule Status Collisions")

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-rule-status-collision-host"},
		OrbitNodeKey: "santa-rule-status-collision-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	labelID := createSantaRuleLabel(t, db, "Santa Rule Status Collisions")
	if err := labelStore.SetMembership(ctx, labelID, host.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}

	_, firstBundleHash, firstSHA, secondSHA := createSantaRuleBundleFixture(t, db)
	secondBundleID, secondBundleHash := createSantaRuleBundleWithExecutable(
		t,
		db,
		"Duplicate Bundle Rule App",
		"Bundle Rule App",
		firstSHA,
	)
	for name, bundleHash := range map[string]string{
		"First bundle":     firstBundleHash,
		"Duplicate bundle": secondBundleHash,
	} {
		if _, err := ruleStore.Create(ctx, rules.RuleMutation{
			ConfigurationID: configurationID,
			RuleType:        rules.RuleTypeBundle,
			Identifier:      bundleHash,
			Name:            name,
			Policy:          rules.PolicyBlocklist,
			Targets:         storeRuleTargets([]int64{labelID}),
		}); err != nil {
			t.Fatalf("create %s rule: %v", name, err)
		}
	}

	resolved, err := ruleStore.ResolveRulesForHost(ctx, host.ID, configurationID)
	if err != nil {
		t.Fatalf("resolve duplicate bundle rules: %v", err)
	}
	if len(resolved) != 3 {
		t.Fatalf("resolved rules = %d, want three pre-canonical bundle expansions", len(resolved))
	}
	targets, err := rules.SyncTargetsFromRules(resolved)
	if err != nil {
		t.Fatalf("canonicalize identical bundle rules: %v", err)
	}
	if len(targets) != 2 || targets[0].Identifier != firstSHA || targets[1].Identifier != secondSHA {
		t.Fatalf("canonical targets = %+v, want two unique bundle executables", targets)
	}

	syncStore := syncstate.NewStore(db)
	syncType, err := syncStore.PreparePending(
		ctx,
		host.ID,
		targets,
		syncstate.RuleCounts{},
		false,
		strings.Repeat("0", 32),
	)
	if err != nil {
		t.Fatalf("prepare canonical targets: %v", err)
	}
	if err := syncStore.PromotePending(
		ctx,
		host.ID,
		2,
		2,
		syncType,
		strings.Repeat("1", 32),
	); err != nil {
		t.Fatalf("promote canonical targets: %v", err)
	}

	for pageIndex, wantIdentifier := range []string{firstSHA, secondSHA} {
		statuses, count, err := ruleStore.ListRuleStatusesForHost(
			ctx,
			host.ID,
			configurationID,
			rules.RuleStatusListParams{ListParams: dbutil.ListParams{
				PageIndex: int32(pageIndex),
				PageSize:  1,
			}},
		)
		if err != nil {
			t.Fatalf("list canonical rule status page %d: %v", pageIndex, err)
		}
		if count != 2 || len(statuses) != 1 || statuses[0].Identifier != wantIdentifier || !statuses[0].Applied {
			t.Fatalf("status page %d = %+v count %d, want applied %q of two", pageIndex, statuses, count, wantIdentifier)
		}
	}

	if _, err := db.Pool().Exec(
		ctx,
		`UPDATE santa_bundles SET name = 'Different App Name' WHERE id = $1`,
		secondBundleID,
	); err != nil {
		t.Fatalf("change duplicate bundle app name: %v", err)
	}
	resolved, err = ruleStore.ResolveRulesForHost(ctx, host.ID, configurationID)
	if err != nil {
		t.Fatalf("resolve app-name conflict: %v", err)
	}
	if _, err := rules.SyncTargetsFromRules(resolved); !errors.Is(err, dbutil.ErrConflict) {
		t.Fatalf("app-name-only collision error = %v, want ErrConflict", err)
	}
	if _, _, err := ruleStore.ListRuleStatusesForHost(
		ctx,
		host.ID,
		configurationID,
		rules.RuleStatusListParams{ListParams: dbutil.ListParams{PageSize: 1}},
	); !errors.Is(err, dbutil.ErrConflict) {
		t.Fatalf("status app-name-only collision error = %v, want ErrConflict", err)
	}
}

func TestBundleRuleExpandsToBinaryHostRules(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := rules.NewStore(db)
	configurationID := createSantaRuleConfiguration(t, db, "Santa Bundle Rule")

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

	bundleID, bundleHash, firstSHA, secondSHA := createSantaRuleBundleFixture(t, db)

	rule, err := store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeBundle,
		Identifier:      bundleHash,
		Name:            "Bundle Rule",
		Policy:          rules.PolicyBlocklist,
		Targets:         storeRuleTargets([]int64{labelID}),
	})
	if err != nil {
		t.Fatalf("create bundle rule: %v", err)
	}

	got, err := store.ResolveRulesForHost(ctx, host.ID, configurationID)
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

	targets, err := rules.SyncTargetsFromRules(got)
	if err != nil {
		t.Fatalf("build sync targets: %v", err)
	}
	if len(targets) != 2 || targets[0].RuleType != string(rules.RuleTypeBinary) ||
		targets[0].AppName != "Bundle Rule App" {
		t.Fatalf("sync targets = %+v, want binary payloads carrying bundle notification data", targets)
	}

	if _, err := db.Pool().Exec(ctx, `UPDATE santa_bundles SET name = '' WHERE id = $1`, bundleID); err != nil {
		t.Fatalf("clear bundle name: %v", err)
	}
	got, err = store.ResolveRulesForHost(ctx, host.ID, configurationID)
	if err != nil {
		t.Fatalf("resolve unnamed bundle rule: %v", err)
	}
	for _, hostRule := range got {
		if hostRule.AppName != "" {
			t.Fatalf("unnamed bundle notification app name = %q, want empty", hostRule.AppName)
		}
	}

	if _, err := store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeBinary,
		Identifier:      firstSHA,
		Name:            "Conflicting binary",
		Policy:          rules.PolicyAllowlist,
		Targets:         storeRuleTargets([]int64{labelID}),
	}); err != nil {
		t.Fatalf("create conflicting direct rule: %v", err)
	}
	got, err = store.ResolveRulesForHost(ctx, host.ID, configurationID)
	if err != nil {
		t.Fatalf("resolve conflicting rules: %v", err)
	}
	if _, err := rules.SyncTargetsFromRules(got); !errors.Is(err, dbutil.ErrConflict) {
		t.Fatalf("conflicting bundle expansion error = %v, want ErrConflict", err)
	}
}

func TestRuleStoreBulkDeleteIgnoresMissingIDs(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)
	configurationID := createSantaRuleConfiguration(t, db, "Santa Bulk Delete")

	first, err := store.Create(
		ctx,
		rules.RuleMutation{
			ConfigurationID: configurationID,
			RuleType:        rules.RuleTypeBinary,
			Identifier:      strings.Repeat("3", 64),
			Name:            "Bulk Binary",
			Policy:          rules.PolicyAllowlist,
		},
	)
	if err != nil {
		t.Fatalf("create first rule: %v", err)
	}
	second, err := store.Create(ctx, rules.RuleMutation{
		ConfigurationID: configurationID,
		RuleType:        rules.RuleTypeTeamID,
		Identifier:      "BULKTEAM12",
		Name:            "Bulk Team",
		Policy:          rules.PolicyAllowlist,
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

func TestRuleStoreListsMultipleRuleTypes(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)
	configurationID := createSantaRuleConfiguration(t, db, "Santa Rule Filters")
	for _, mutation := range []rules.RuleMutation{
		{ConfigurationID: configurationID, RuleType: rules.RuleTypeBinary, Identifier: strings.Repeat("4", 64), Name: "Binary", Policy: rules.PolicyAllowlist},
		{ConfigurationID: configurationID, RuleType: rules.RuleTypeTeamID, Identifier: "FILTERTEAM", Name: "Team", Policy: rules.PolicyAllowlist},
	} {
		if _, err := store.Create(ctx, mutation); err != nil {
			t.Fatalf("create %s rule: %v", mutation.RuleType, err)
		}
	}

	got, count, err := store.List(ctx, rules.RuleListParams{
		ConfigurationIDs: []int64{configurationID},
		RuleTypes:        []rules.RuleType{rules.RuleTypeBinary, rules.RuleTypeTeamID},
	})
	if err != nil {
		t.Fatalf("list multiple rule types: %v", err)
	}
	if count != 2 || len(got) != 2 {
		t.Fatalf("rules = %+v count=%d, want binary and team ID rules", got, count)
	}
}

func TestRuleIdentityIsUniqueWithinConfiguration(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := rules.NewStore(db)
	host, err := hosts.NewStore(db).UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-rule-configuration-boundary"},
		OrbitNodeKey: "santa-rule-configuration-boundary-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	labelID := createSantaRuleLabel(t, db, "Santa Configuration Boundary")
	if err := labels.NewStore(db).SetMembership(ctx, labelID, host.ID, true); err != nil {
		t.Fatalf("set host label: %v", err)
	}
	firstConfigurationID := createSantaRuleConfiguration(t, db, "Normal Macs")
	secondConfigurationID := createSantaRuleConfiguration(t, db, "SAC Macs")
	identifier := "ABCDE12345:com.google.Chrome"

	for _, mutation := range []rules.RuleMutation{
		{
			ConfigurationID: firstConfigurationID,
			RuleType:        rules.RuleTypeSigningID,
			Identifier:      identifier,
			Name:            "Allow Chrome",
			Policy:          rules.PolicyAllowlist,
			Targets:         storeRuleTargets([]int64{labelID}),
		},
		{
			ConfigurationID: secondConfigurationID,
			RuleType:        rules.RuleTypeSigningID,
			Identifier:      identifier,
			Name:            "Block Chrome",
			Policy:          rules.PolicyBlocklist,
			Targets:         storeRuleTargets([]int64{labelID}),
		},
	} {
		if _, err := store.Create(ctx, mutation); err != nil {
			t.Fatalf("create %q: %v", mutation.Name, err)
		}
	}
	for configurationID, wantPolicy := range map[int64]rules.Policy{
		firstConfigurationID:  rules.PolicyAllowlist,
		secondConfigurationID: rules.PolicyBlocklist,
	} {
		resolved, err := store.ResolveRulesForHost(ctx, host.ID, configurationID)
		if err != nil {
			t.Fatalf("resolve configuration %d: %v", configurationID, err)
		}
		if len(resolved) != 1 || resolved[0].Policy != wantPolicy {
			t.Fatalf("configuration %d rules = %+v, want %s Chrome", configurationID, resolved, wantPolicy)
		}
	}

	_, err = store.Create(ctx, rules.RuleMutation{
		ConfigurationID: secondConfigurationID,
		RuleType:        rules.RuleTypeSigningID,
		Identifier:      identifier,
		Name:            "Duplicate SAC Chrome",
		Policy:          rules.PolicySilentBlocklist,
	})
	if !errors.Is(err, dbutil.ErrAlreadyExists) {
		t.Fatalf("duplicate rule error = %v, want ErrAlreadyExists", err)
	}
}

func createSantaRuleBundleFixture(t *testing.T, db *database.DB) (int64, string, string, string) {
	t.Helper()

	ctx := t.Context()
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
	return bundleID, bundleHash, firstSHA, secondSHA
}

func createSantaRuleBundleWithExecutable(
	t *testing.T,
	db *database.DB,
	bundleID string,
	name string,
	executableSHA string,
) (int64, string) {
	t.Helper()

	ctx := t.Context()
	bundleHash := strings.Repeat("c", 64)
	var databaseBundleID int64
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
		VALUES ($1, $2, $3, '/Applications/Duplicate Bundle Rule.app', '1', '1', 1, now())
		RETURNING id
	`, bundleHash, bundleID, name).Scan(&databaseBundleID); err != nil {
		t.Fatalf("insert duplicate bundle: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_bundle_executables (bundle_id, executable_id)
		SELECT $1, id
		FROM santa_executables
		WHERE sha256 = $2
	`, databaseBundleID, executableSHA); err != nil {
		t.Fatalf("link duplicate bundle executable: %v", err)
	}
	return databaseBundleID, bundleHash
}

func createSantaRuleConfiguration(t *testing.T, db *database.DB, name string) int64 {
	t.Helper()

	configuration, err := configurations.NewStore(db).Create(t.Context(), configurations.ConfigurationMutation{
		Name:                     name,
		ClientMode:               configurations.ClientModeMonitor,
		OverrideFileAccessAction: configurations.FileAccessActionNone,
		FullSyncIntervalSeconds:  600,
		BatchSize:                50,
	})
	if err != nil {
		t.Fatalf("create configuration %q: %v", name, err)
	}
	return configuration.ID
}

func storeRuleTargets(includeLabelIDs []int64, excludedLabelIDs ...int64) rules.RuleTargets {
	return rules.RuleTargets{
		Include: santaRuleLabelRefs(includeLabelIDs...),
		Exclude: santaRuleLabelRefs(excludedLabelIDs...),
	}
}

func santaRuleLabelRefs(labelIDs ...int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(labelIDs))
	for i, labelID := range labelIDs {
		refs[i] = targeting.LabelRef{LabelID: labelID}
	}
	return refs
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
