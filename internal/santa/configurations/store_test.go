package configurations_test

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// baseline returns a valid ConfigurationMutation using Santa's own defaults.
func baseline(name string) configurations.ConfigurationMutation {
	return configurations.ConfigurationMutation{
		Name:                     name,
		ClientMode:               configurations.ClientModeMonitor,
		OverrideFileAccessAction: configurations.FileAccessActionNone,
		FullSyncIntervalSeconds:  600,
		BatchSize:                50,
	}
}

func TestConfigurationStoreValidatesConflictsAndReplacesEditableShape(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := configurations.NewStore(db)
	firstLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration First")
	secondLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration Second")
	thirdLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration Third")

	short := baseline("short sync")
	short.FullSyncIntervalSeconds = 59
	if _, err := store.Create(ctx, short); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("short full sync error = %v, want ErrInvalidInput", err)
	}

	tinyBatch := baseline("tiny batch")
	tinyBatch.BatchSize = 1
	if _, err := store.Create(ctx, tinyBatch); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("tiny batch size error = %v, want ErrInvalidInput", err)
	}

	missingName := baseline("")
	if _, err := store.Create(ctx, missingName); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("missing name error = %v, want ErrInvalidInput", err)
	}

	emptyClientMode := baseline("empty client mode")
	emptyClientMode.ClientMode = ""
	if _, err := store.Create(ctx, emptyClientMode); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("empty client mode error = %v, want ErrInvalidInput", err)
	}

	invalidFileAccessAction := baseline("invalid file access action")
	invalidFileAccessAction.OverrideFileAccessAction = ""
	if _, err := store.Create(ctx, invalidFileAccessAction); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("invalid file access action error = %v, want ErrInvalidInput", err)
	}

	invalidLabel := baseline("invalid label")
	invalidLabel.Targets = configurationTargets(labelRefs(0), nil)
	if _, err := store.Create(ctx, invalidLabel); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("invalid label ID error = %v, want ErrInvalidInput", err)
	}

	missingLabel := baseline("missing label")
	missingLabel.Targets = configurationTargets(labelRefs(999_999), nil)
	if _, err := store.Create(ctx, missingLabel); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("missing label ID error = %v, want ErrNotFound", err)
	}

	duplicateTargets := baseline("duplicate targets")
	duplicateTargets.Targets = configurationTargets(labelRefs(firstLabelID, firstLabelID), nil)
	if _, err := store.Create(ctx, duplicateTargets); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("duplicate target error = %v, want ErrInvalidInput", err)
	}

	overlappingTargets := baseline("overlapping targets")
	overlappingTargets.Targets = configurationTargets(labelRefs(firstLabelID), labelRefs(firstLabelID))
	if _, err := store.Create(ctx, overlappingTargets); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("overlapping target error = %v, want ErrInvalidInput", err)
	}

	remountWithoutFlags := baseline("remount without flags")
	remountWithoutFlags.RemovableMediaPolicy = configurations.RemovableMediaPolicy{
		Action: configurations.RemovableMediaActionRemount,
	}
	if _, err := store.Create(ctx, remountWithoutFlags); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("remount without flags error = %v, want ErrInvalidInput", err)
	}

	create := baseline("Baseline")
	create.Description = "Baseline policy"
	create.ClientMode = configurations.ClientModeLockdown
	create.EnableBundles = true
	create.DisableUnknownEventUpload = true
	create.OverrideFileAccessAction = configurations.FileAccessActionDisable
	create.FullSyncIntervalSeconds = 120
	create.RemovableMediaPolicy = configurations.RemovableMediaPolicy{
		Action:       configurations.RemovableMediaActionRemount,
		RemountFlags: []string{"rw", "nosuid"},
	}
	create.Targets = configurationTargets(labelRefs(firstLabelID, secondLabelID), labelRefs(thirdLabelID))

	config, err := store.Create(ctx, create)
	if err != nil {
		t.Fatalf("create configuration: %v", err)
	}
	if config.Name != "Baseline" || config.Description != "Baseline policy" ||
		config.Position != 0 || config.ClientMode != configurations.ClientModeLockdown {
		t.Fatalf("configuration = %+v, want baseline lockdown policy", config)
	}
	if !config.EnableBundles ||
		!config.DisableUnknownEventUpload ||
		config.OverrideFileAccessAction != configurations.FileAccessActionDisable ||
		len(config.RemovableMediaPolicy.RemountFlags) != 2 {
		t.Fatalf("settings were not preserved: %+v", config)
	}
	if !sameLabelRefs(config.Targets.Include, labelRefs(firstLabelID, secondLabelID)) ||
		!sameLabelRefs(config.Targets.Exclude, labelRefs(thirdLabelID)) {
		t.Fatalf("targets = %+v, want include [%d %d] exclude [%d]",
			config.Targets, firstLabelID, secondLabelID, thirdLabelID)
	}

	overlapping := baseline("Overlapping")
	overlapping.Targets = configurationTargets(labelRefs(firstLabelID), nil)
	if _, err := store.Create(ctx, overlapping); err != nil {
		t.Fatalf("overlapping configuration create error = %v, want allowed overlap", err)
	}

	update := baseline("Updated")
	update.Description = "Updated policy"
	update.Targets = configurationTargets(labelRefs(thirdLabelID), nil)
	updated, err := store.Update(ctx, config.ID, update)
	if err != nil {
		t.Fatalf("update configuration: %v", err)
	}
	if updated.Name != "Updated" || updated.Description != "Updated policy" ||
		updated.ClientMode != configurations.ClientModeMonitor {
		t.Fatalf("updated configuration = %+v", updated)
	}
	if updated.EnableBundles ||
		updated.DisableUnknownEventUpload ||
		updated.OverrideFileAccessAction != configurations.FileAccessActionNone ||
		!updated.RemovableMediaPolicy.IsZero() {
		t.Fatalf("update did not replace settings: %+v", updated)
	}
	if !sameLabelRefs(updated.Targets.Include, labelRefs(thirdLabelID)) || len(updated.Targets.Exclude) != 0 {
		t.Fatalf("updated targets = %+v, want only include label %d", updated.Targets, thirdLabelID)
	}
}

func TestConfigurationResolverUsesFirstMatchingPosition(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := configurations.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-configuration-resolver-host"},
		OrbitNodeKey: "santa-configuration-resolver-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	firstLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration Resolver First")
	secondLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration Resolver Second")
	if err := labelStore.SetMembership(ctx, firstLabelID, host.ID, true); err != nil {
		t.Fatalf("set first label membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, secondLabelID, host.ID, true); err != nil {
		t.Fatalf("set second label membership: %v", err)
	}

	first := baseline("First")
	first.Targets = configurationTargets(labelRefs(secondLabelID, firstLabelID), nil)
	firstConfig, err := store.Create(ctx, first)
	if err != nil {
		t.Fatalf("create first configuration: %v", err)
	}
	second := baseline("Second")
	second.Targets = configurationTargets(labelRefs(firstLabelID), nil)
	secondConfig, err := store.Create(ctx, second)
	if err != nil {
		t.Fatalf("create second configuration: %v", err)
	}

	resolved, err := store.ResolveConfigurationForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve configuration: %v", err)
	}
	if resolved == nil || resolved.ID != firstConfig.ID || resolved.MatchedViaLabel == nil ||
		resolved.MatchedViaLabel.ID != secondLabelID {
		t.Fatalf("resolved configuration = %+v, want first configuration via second label", resolved)
	}
	if len(resolved.Targets.Include) != 0 || len(resolved.Targets.Exclude) != 0 {
		t.Fatalf("resolved targets = %+v, want light resolver without hydrated targets", resolved.Targets)
	}

	resolvedWithTargets, err := store.ResolveConfigurationForHostWithTargets(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve configuration with targets: %v", err)
	}
	if resolvedWithTargets == nil ||
		!sameLabelRefs(resolvedWithTargets.Targets.Include, labelRefs(secondLabelID, firstLabelID)) ||
		len(resolvedWithTargets.Targets.Exclude) != 0 {
		t.Fatalf("resolved targets = %+v, want first configuration target set", resolvedWithTargets)
	}

	if err := store.ReorderConfigurations(ctx, []int64{secondConfig.ID, firstConfig.ID}); err != nil {
		t.Fatalf("reorder configurations: %v", err)
	}
	resolved, err = store.ResolveConfigurationForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve configuration after reorder: %v", err)
	}
	if resolved == nil || resolved.ID != secondConfig.ID || resolved.MatchedViaLabel == nil ||
		resolved.MatchedViaLabel.ID != firstLabelID {
		t.Fatalf("resolved configuration after reorder = %+v, want second configuration", resolved)
	}

	err = store.ReorderConfigurations(ctx, []int64{secondConfig.ID})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("partial reorder error = %v, want ErrInvalidInput", err)
	}

	missing, err := store.ResolveConfigurationForHost(ctx, host.ID+9999)
	if err != nil {
		t.Fatalf("resolve missing host configuration: %v", err)
	}
	if missing != nil {
		t.Fatalf("missing host configuration = %+v, want nil", missing)
	}
}

func TestConfigurationResolverUsesExclusions(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := configurations.NewStore(db)

	studentHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-configuration-student-host"},
		OrbitNodeKey: "santa-configuration-student-orbit",
	})
	if err != nil {
		t.Fatalf("enroll student host: %v", err)
	}
	sacHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-configuration-sac-host"},
		OrbitNodeKey: "santa-configuration-sac-orbit",
	})
	if err != nil {
		t.Fatalf("enroll sac host: %v", err)
	}

	allStudentsID := createSantaConfigurationLabel(t, db, "Santa Configuration All Students")
	sacID := createSantaConfigurationLabel(t, db, "Santa Configuration SAC")
	if err := labelStore.SetMembership(ctx, allStudentsID, studentHost.ID, true); err != nil {
		t.Fatalf("set student all-students membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, allStudentsID, sacHost.ID, true); err != nil {
		t.Fatalf("set sac all-students membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, sacID, sacHost.ID, true); err != nil {
		t.Fatalf("set sac membership: %v", err)
	}

	broad := baseline("All Students except SAC")
	broad.Targets = configurationTargets(labelRefs(allStudentsID), labelRefs(sacID))
	broadConfig, err := store.Create(ctx, broad)
	if err != nil {
		t.Fatalf("create broad configuration: %v", err)
	}
	narrow := baseline("SAC")
	narrow.Targets = configurationTargets(labelRefs(sacID), nil)
	narrowConfig, err := store.Create(ctx, narrow)
	if err != nil {
		t.Fatalf("create narrow configuration: %v", err)
	}

	resolved, err := store.ResolveConfigurationForHost(ctx, studentHost.ID)
	if err != nil {
		t.Fatalf("resolve student configuration: %v", err)
	}
	if resolved == nil || resolved.ID != broadConfig.ID {
		t.Fatalf("student resolved configuration = %+v, want broad config", resolved)
	}

	resolved, err = store.ResolveConfigurationForHost(ctx, sacHost.ID)
	if err != nil {
		t.Fatalf("resolve sac configuration: %v", err)
	}
	if resolved == nil || resolved.ID != narrowConfig.ID {
		t.Fatalf("sac resolved configuration = %+v, want narrow config", resolved)
	}
}

func TestConfigurationResolverRequiresIncludeTarget(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := configurations.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-configuration-requires-include-host"},
		OrbitNodeKey: "santa-configuration-requires-include-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	excludedID := createSantaConfigurationLabel(t, db, "Santa Configuration Exclude Only")
	if err := labelStore.SetMembership(ctx, excludedID, host.ID, true); err != nil {
		t.Fatalf("set excluded membership: %v", err)
	}

	excludeOnly := baseline("Exclude only")
	excludeOnly.Targets = configurationTargets(nil, labelRefs(excludedID))
	if _, err := store.Create(ctx, excludeOnly); err != nil {
		t.Fatalf("create exclude-only configuration: %v", err)
	}

	resolved, err := store.ResolveConfigurationForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve configuration: %v", err)
	}
	if resolved != nil {
		t.Fatalf("resolved configuration = %+v, want nil", resolved)
	}
}

func TestConfigurationStoreBulkDeleteIgnoresMissingIDs(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := configurations.NewStore(db)

	first, err := store.Create(ctx, baseline("First"))
	if err != nil {
		t.Fatalf("create first configuration: %v", err)
	}
	second, err := store.Create(ctx, baseline("Second"))
	if err != nil {
		t.Fatalf("create second configuration: %v", err)
	}

	deleted, err := store.DeleteMany(ctx, []int64{first.ID, second.ID + 999})
	if err != nil {
		t.Fatalf("bulk delete configurations: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}

	got, _, err := store.List(ctx, configurations.ConfigurationListParams{})
	if err != nil {
		t.Fatalf("list configurations: %v", err)
	}
	if len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("configurations after delete = %+v, want only second", got)
	}
}

func createSantaConfigurationLabel(t *testing.T, db *database.DB, name string) int64 {
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

func configurationTargets(include, exclude []targeting.LabelRef) configurations.ConfigurationTargets {
	return configurations.ConfigurationTargets{Include: include, Exclude: exclude}
}

func labelRefs(labelIDs ...int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(labelIDs))
	for i, labelID := range labelIDs {
		refs[i] = targeting.LabelRef{LabelID: labelID}
	}
	return refs
}

func sameLabelRefs(got, want []targeting.LabelRef) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i].LabelID != want[i].LabelID {
			return false
		}
	}
	return true
}
