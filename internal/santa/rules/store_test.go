package rules_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestRuleStoreValidatesAndReplacesEditableShape(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := rules.NewStore(db)
	labelID := createSantaRuleLabel(t, db, "Santa Rule Validation")
	replacementLabelID := createSantaRuleLabel(t, db, "Santa Rule Replacement")
	excludeLabelID := createSantaRuleLabel(t, db, "Santa Rule Exclude")
	allHostsLabelID := santaRuleAllHostsLabelID(t, db)
	binaryIdentifier := strings.Repeat("a", 64)

	invalidCases := []struct {
		name   string
		params rules.RuleMutation
	}{
		{name: "missing type", params: rules.RuleMutation{Name: "Missing Type", Identifier: binaryIdentifier}},
		{
			name:   "missing identifier",
			params: rules.RuleMutation{Name: "Missing Identifier", RuleType: rules.RuleTypeBinary},
		},
		{
			name: "missing name",
			params: rules.RuleMutation{
				RuleType:   rules.RuleTypeBinary,
				Identifier: binaryIdentifier,
			},
		},
		{
			name: "cel without expression",
			params: rules.RuleMutation{
				RuleType:   rules.RuleTypeBinary,
				Identifier: binaryIdentifier,
				Name:       "CEL Without Expression",
				Targets:    ruleTargets([]rules.RuleInclude{{Policy: rules.PolicyCEL, LabelID: labelID}}),
			},
		},
		{
			name: "non cel with expression",
			params: rules.RuleMutation{
				RuleType:   rules.RuleTypeBinary,
				Identifier: binaryIdentifier,
				Name:       "Non CEL With Expression",
				Targets: ruleTargets([]rules.RuleInclude{{
					Policy:        rules.PolicyAllowlist,
					CELExpression: "target.path == '/Applications'",
					LabelID:       labelID,
				}}),
			},
		},
		{
			name: "duplicate include label",
			params: rules.RuleMutation{
				RuleType:   rules.RuleTypeBinary,
				Identifier: binaryIdentifier,
				Name:       "Duplicate Include Label",
				Targets: ruleTargets([]rules.RuleInclude{
					{Policy: rules.PolicyAllowlist, LabelID: labelID},
					{Policy: rules.PolicyBlocklist, LabelID: labelID},
				}),
			},
		},
		{
			name: "duplicate exclude label",
			params: rules.RuleMutation{
				RuleType:   rules.RuleTypeBinary,
				Identifier: binaryIdentifier,
				Name:       "Duplicate Exclude Label",
				Targets: ruleTargets(
					[]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: labelID}},
					excludeLabelID,
					excludeLabelID,
				),
			},
		},
		{
			name: "include and exclude overlap",
			params: rules.RuleMutation{
				RuleType:   rules.RuleTypeBinary,
				Identifier: binaryIdentifier,
				Name:       "Include Exclude Overlap",
				Targets: ruleTargets(
					[]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: labelID}},
					labelID,
				),
			},
		},
		{
			name: "builtin exclude",
			params: rules.RuleMutation{
				RuleType:   rules.RuleTypeBinary,
				Identifier: binaryIdentifier,
				Name:       "Builtin Exclude",
				Targets: ruleTargets(
					[]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: labelID}},
					allHostsLabelID,
				),
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

	rule, err := store.CreateRule(ctx, rules.RuleMutation{
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

	_, err = store.CreateRule(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: binaryIdentifier,
		Name:       "Duplicate",
	})
	if !errors.Is(err, dbutil.ErrAlreadyExists) {
		t.Fatalf("duplicate CreateRule error = %v, want ErrAlreadyExists", err)
	}

	celExpression := "target.path.startsWith('/Applications')"
	updated, err := store.UpdateRule(ctx, rule.ID, rules.RuleMutation{
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
	db, ctx := dbtest.Open(t)
	store := rules.NewStore(db)

	_, err := store.CreateRule(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: strings.Repeat("d", 64),
		Name:       "Missing Include Label",
		Targets:    ruleTargets([]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: 999_999}}),
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("missing include label error = %v, want ErrNotFound", err)
	}

	labelID := createSantaRuleLabel(t, db, "Rule Missing Exclude Include")
	_, err = store.CreateRule(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Identifier: strings.Repeat("e", 64),
		Name:       "Missing Exclude Label",
		Targets:    ruleTargets([]rules.RuleInclude{{Policy: rules.PolicyAllowlist, LabelID: labelID}}, 999_999),
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("missing exclude label error = %v, want ErrNotFound", err)
	}
}

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

func TestRuleResolverUsesExcludeAndIncludePriority(t *testing.T) {
	db, ctx := dbtest.Open(t)
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

	hostRule, err := store.CreateRule(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBinary,
		Name:       "Scoped Binary",
		Identifier: strings.Repeat("1", 64),
		Targets: ruleTargets([]rules.RuleInclude{
			{Policy: rules.PolicyBlocklist, LabelID: firstLabelID},
			{Policy: rules.PolicySilentBlocklist, LabelID: secondLabelID},
		}),
	})
	if err != nil {
		t.Fatalf("create host rule: %v", err)
	}
	excludedRule, err := store.CreateRule(ctx, rules.RuleMutation{
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
	if got[0].RuleID != hostRule.ID || got[0].Name != "Scoped Binary" ||
		got[0].Policy != rules.PolicySilentBlocklist {
		t.Fatalf("host rule = %+v, want second include to win", got[0])
	}
	if got[0].RuleID == excludedRule.ID {
		t.Fatalf("excluded rule resolved: %+v", got[0])
	}
}

func TestRuleResolverAllowsAllHostsInclude(t *testing.T) {
	db, ctx := dbtest.Open(t)
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

	rule, err := store.CreateRule(ctx, rules.RuleMutation{
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

func TestBundleRuleExpandsToBinaryHostRules(t *testing.T) {
	db, ctx := dbtest.Open(t)
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

	rule, err := store.CreateRule(ctx, rules.RuleMutation{
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
}

func TestRuleTargetsSearchBundlesAndSoftwareInventory(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := rules.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-rule-target-software-host"},
		OrbitNodeKey: "santa-rule-target-software-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	completeBundleHash := strings.Repeat("c", 64)
	incompleteBundleHash := strings.Repeat("d", 64)
	var executableID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_executables (sha256, file_name, file_bundle_name)
		VALUES ($1, 'Target Main', 'Target Bundle')
		RETURNING id
	`, strings.Repeat("3", 64)).Scan(&executableID); err != nil {
		t.Fatalf("insert executable: %v", err)
	}
	var completeBundleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_bundles (sha256, bundle_id, name, path, version, binary_count, uploaded_at)
		VALUES ($1, 'com.example.target', 'Target Bundle', '/Applications/Target.app', '1.0', 1, now())
		RETURNING id
	`, completeBundleHash).Scan(&completeBundleID); err != nil {
		t.Fatalf("insert complete bundle: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_bundle_executables (bundle_id, executable_id)
		VALUES ($1, $2)
	`, completeBundleID, executableID); err != nil {
		t.Fatalf("link complete bundle: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_bundles (sha256, bundle_id, name, path, version, binary_count)
		VALUES ($1, 'com.example.incomplete', 'Incomplete Bundle', '/Applications/Incomplete.app', '1.0', 2)
	`, incompleteBundleHash); err != nil {
		t.Fatalf("insert incomplete bundle: %v", err)
	}

	targets, err := store.ListRuleTargets(ctx, rules.RuleTargetListParams{
		Q:          "Target Bundle",
		TargetType: rules.RuleTypeBundle,
	})
	if err != nil {
		t.Fatalf("list bundle targets: %v", err)
	}
	if len(targets) != 1 ||
		targets[0].Identifier != completeBundleHash ||
		targets[0].DisplayName != "Target Bundle" ||
		targets[0].BundleIdentifier != "com.example.target" ||
		targets[0].BinaryCount != 1 ||
		targets[0].CollectedBinaryCount != 1 ||
		!targets[0].Complete {
		t.Fatalf("bundle targets = %+v, want complete bundle candidate", targets)
	}

	_, err = store.CreateRule(ctx, rules.RuleMutation{
		RuleType:   rules.RuleTypeBundle,
		Identifier: incompleteBundleHash,
		Name:       "Incomplete Bundle",
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("incomplete bundle CreateRule error = %v, want ErrInvalidInput", err)
	}

	var titleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO software_titles (name, display_name, source, bundle_identifier)
		VALUES ('Software Target', 'Software Target', 'apps', 'com.example.software')
		RETURNING id
	`).Scan(&titleID); err != nil {
		t.Fatalf("insert software title: %v", err)
	}
	var softwareID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO software (title_id, name, version, source, bundle_identifier)
		VALUES ($1, 'Software Target', '9.8.7', 'apps', 'com.example.software')
		RETURNING id
	`, titleID).Scan(&softwareID); err != nil {
		t.Fatalf("insert software: %v", err)
	}
	softwareHash := strings.Repeat("4", 64)
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO host_software_installed_paths (
			host_id,
			software_id,
			installed_path,
			team_identifier,
			cdhash_sha256,
			executable_sha256,
			executable_path
		)
		VALUES ($1, $2, '/Applications/Software Target.app', 'TEAMSOFT12', 'soft-cdhash', $3, '/Applications/Software Target.app/Contents/MacOS/Software Target')
	`, host.ID, softwareID, softwareHash); err != nil {
		t.Fatalf("insert software path: %v", err)
	}

	softwareTargets, err := store.ListRuleTargets(ctx, rules.RuleTargetListParams{
		Q:          softwareHash,
		TargetType: rules.RuleTypeBinary,
	})
	if err != nil {
		t.Fatalf("list software-backed targets: %v", err)
	}
	if len(softwareTargets) != 1 ||
		softwareTargets[0].Identifier != softwareHash ||
		softwareTargets[0].DisplayName != "Software Target" ||
		softwareTargets[0].BundleIdentifier != "com.example.software" ||
		softwareTargets[0].Path != "/Applications/Software Target.app/Contents/MacOS/Software Target" {
		t.Fatalf("software targets = %+v, want osquery binary candidate", softwareTargets)
	}

	teamTargets, err := store.ListRuleTargets(ctx, rules.RuleTargetListParams{
		Q:          "TEAMSOFT12",
		TargetType: rules.RuleTypeTeamID,
	})
	if err != nil {
		t.Fatalf("list team targets: %v", err)
	}
	if len(teamTargets) != 1 ||
		teamTargets[0].Identifier != "TEAMSOFT12" ||
		teamTargets[0].DisplayName != "" ||
		teamTargets[0].BundleIdentifier != "com.example.software" {
		t.Fatalf("team targets = %+v, want publisher identity without app display fallback", teamTargets)
	}

	signingTargets, err := store.ListRuleTargets(ctx, rules.RuleTargetListParams{
		Q:          "TEAMSOFT12:com.example.software",
		TargetType: rules.RuleTypeSigningID,
	})
	if err != nil {
		t.Fatalf("list signing targets: %v", err)
	}
	if len(signingTargets) != 1 ||
		signingTargets[0].Identifier != "TEAMSOFT12:com.example.software" ||
		signingTargets[0].DisplayName != "Software Target" {
		t.Fatalf("signing targets = %+v, want signing ID plus software display name", signingTargets)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_executables (
			sha256,
			file_name,
			file_bundle_name,
			file_bundle_id,
			team_id,
			signing_id
		)
		VALUES ($1, 'Software Target Helper', 'Observed Software Target', 'com.example.software', 'TEAMSOFT12', 'TEAMSOFT12:com.example.software')
	`, strings.Repeat("7", 64)); err != nil {
		t.Fatalf("insert observed signing target: %v", err)
	}
	ambiguousSigningTargets, err := store.ListRuleTargets(ctx, rules.RuleTargetListParams{
		Q:          "Observed Software Target",
		TargetType: rules.RuleTypeSigningID,
	})
	if err != nil {
		t.Fatalf("list ambiguous signing targets: %v", err)
	}
	if len(ambiguousSigningTargets) != 1 ||
		ambiguousSigningTargets[0].Identifier != "TEAMSOFT12:com.example.software" ||
		ambiguousSigningTargets[0].DisplayName != "" ||
		ambiguousSigningTargets[0].BundleIdentifier != "com.example.software" {
		t.Fatalf(
			"ambiguous signing targets = %+v, want searchable context without display fallback",
			ambiguousSigningTargets,
		)
	}

	certificateHash := strings.Repeat("6", 64)
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_certificates (sha256, common_name, organization, organizational_unit)
		VALUES ($1, 'Developer ID Application: Example', 'Example Org', 'TEAMSOFT12')
	`, certificateHash); err != nil {
		t.Fatalf("insert certificate: %v", err)
	}
	certificateTargets, err := store.ListRuleTargets(ctx, rules.RuleTargetListParams{
		Q:          "Developer ID",
		TargetType: rules.RuleTypeCertificate,
	})
	if err != nil {
		t.Fatalf("list certificate targets: %v", err)
	}
	if len(certificateTargets) != 1 ||
		certificateTargets[0].Identifier != certificateHash ||
		certificateTargets[0].CertificateCommonName != "Developer ID Application: Example" ||
		certificateTargets[0].CertificateOrganization != "Example Org" ||
		certificateTargets[0].CertificateOrganizationalUnit != "TEAMSOFT12" ||
		certificateTargets[0].DisplayName != "" {
		t.Fatalf("certificate targets = %+v, want fingerprint plus certificate common name", certificateTargets)
	}
}

func TestRuleStoreBulkDeleteIgnoresMissingIDs(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := rules.NewStore(db)

	first, err := store.CreateRule(
		ctx,
		rules.RuleMutation{RuleType: rules.RuleTypeBinary, Identifier: strings.Repeat("3", 64), Name: "Bulk Binary"},
	)
	if err != nil {
		t.Fatalf("create first rule: %v", err)
	}
	second, err := store.CreateRule(ctx, rules.RuleMutation{
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
	if _, err := store.GetRuleByID(ctx, first.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("deleted rule lookup error = %v, want ErrNotFound", err)
	}
	if _, err := store.GetRuleByID(ctx, second.ID); err != nil {
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
