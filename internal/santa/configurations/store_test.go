package configurations_test

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

// baseline returns a valid ConfigurationMutation using Santa's own defaults.
func baseline(name string) configurations.ConfigurationMutation {
	return configurations.ConfigurationMutation{
		Name:                    name,
		ClientMode:              configurations.ClientModeMonitor,
		FullSyncIntervalSeconds: 600,
		BatchSize:               50,
	}
}

func TestConfigurationStoreValidatesConflictsAndReplacesEditableShape(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := configurations.NewStore(db)
	firstLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration First")
	secondLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration Second")

	short := baseline("short sync")
	short.FullSyncIntervalSeconds = 59
	if _, err := store.CreateConfiguration(ctx, short); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("short full sync error = %v, want ErrInvalidInput", err)
	}

	tinyBatch := baseline("tiny batch")
	tinyBatch.BatchSize = 1
	if _, err := store.CreateConfiguration(ctx, tinyBatch); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("tiny batch size error = %v, want ErrInvalidInput", err)
	}

	emptyClientMode := baseline("empty client mode")
	emptyClientMode.ClientMode = ""
	if _, err := store.CreateConfiguration(ctx, emptyClientMode); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("empty client mode error = %v, want ErrInvalidInput", err)
	}

	remountWithoutFlags := baseline("remount without flags")
	remountWithoutFlags.RemovableMediaPolicy = configurations.RemovableMediaPolicy{
		Action: configurations.RemovableMediaActionRemount,
	}
	if _, err := store.CreateConfiguration(ctx, remountWithoutFlags); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("remount without flags error = %v, want ErrInvalidInput", err)
	}

	create := baseline(" Baseline ")
	create.ClientMode = configurations.ClientModeLockdown
	create.EnableBundles = true
	create.FullSyncIntervalSeconds = 120
	create.RemovableMediaPolicy = configurations.RemovableMediaPolicy{
		Action:       configurations.RemovableMediaActionRemount,
		RemountFlags: []string{" rw ", "nosuid"},
	}
	create.LabelIDs = []int64{firstLabelID}

	config, err := store.CreateConfiguration(ctx, create)
	if err != nil {
		t.Fatalf("create configuration: %v", err)
	}
	if config.Name != "Baseline" || config.Position != 0 || config.ClientMode != configurations.ClientModeLockdown {
		t.Fatalf("configuration was not cleaned: %+v", config)
	}
	if !config.EnableBundles || len(config.RemovableMediaPolicy.RemountFlags) != 2 {
		t.Fatalf("settings were not preserved: %+v", config)
	}
	if len(config.LabelIDs) != 1 || config.LabelIDs[0] != firstLabelID {
		t.Fatalf("label IDs = %v, want [%d]", config.LabelIDs, firstLabelID)
	}

	conflictingCreate := baseline("Conflicting")
	conflictingCreate.LabelIDs = []int64{firstLabelID}
	_, err = store.CreateConfiguration(ctx, conflictingCreate)
	var conflict *configurations.ConfigurationLabelConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("label conflict error = %v, want ConfigurationLabelConflictError", err)
	}
	if conflict.LabelID != firstLabelID || conflict.ConfigurationID != config.ID ||
		conflict.ConfigurationName != "Baseline" {
		t.Fatalf("conflict = %+v, want existing configuration details", conflict)
	}

	update := baseline(" Updated ")
	update.LabelIDs = []int64{secondLabelID}
	updated, err := store.UpdateConfiguration(ctx, config.ID, update)
	if err != nil {
		t.Fatalf("update configuration: %v", err)
	}
	if updated.Name != "Updated" || updated.ClientMode != configurations.ClientModeMonitor {
		t.Fatalf("updated configuration = %+v", updated)
	}
	if updated.EnableBundles || !updated.RemovableMediaPolicy.IsZero() {
		t.Fatalf("update did not replace settings: %+v", updated)
	}
	if len(updated.LabelIDs) != 1 || updated.LabelIDs[0] != secondLabelID {
		t.Fatalf("updated label IDs = %v, want [%d]", updated.LabelIDs, secondLabelID)
	}
}

func TestConfigurationResolverUsesFirstMatchingPosition(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := configurations.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-configuration-resolver-host",
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
	first.LabelIDs = []int64{firstLabelID}
	firstConfig, err := store.CreateConfiguration(ctx, first)
	if err != nil {
		t.Fatalf("create first configuration: %v", err)
	}
	second := baseline("Second")
	second.LabelIDs = []int64{secondLabelID}
	secondConfig, err := store.CreateConfiguration(ctx, second)
	if err != nil {
		t.Fatalf("create second configuration: %v", err)
	}

	resolved, err := store.ResolveConfigurationForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve configuration: %v", err)
	}
	if resolved == nil || resolved.ID != firstConfig.ID || resolved.MatchedViaLabel == nil ||
		resolved.MatchedViaLabel.ID != firstLabelID {
		t.Fatalf("resolved configuration = %+v, want first configuration", resolved)
	}

	if err := store.ReorderConfigurations(ctx, []int64{secondConfig.ID, firstConfig.ID}); err != nil {
		t.Fatalf("reorder configurations: %v", err)
	}
	resolved, err = store.ResolveConfigurationForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve configuration after reorder: %v", err)
	}
	if resolved == nil || resolved.ID != secondConfig.ID || resolved.MatchedViaLabel == nil ||
		resolved.MatchedViaLabel.ID != secondLabelID {
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

func TestConfigurationStoreBulkDeleteIgnoresMissingIDs(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := configurations.NewStore(db)

	first, err := store.CreateConfiguration(ctx, baseline("First"))
	if err != nil {
		t.Fatalf("create first configuration: %v", err)
	}
	second, err := store.CreateConfiguration(ctx, baseline("Second"))
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

	got, _, err := store.ListConfigurations(ctx, configurations.ConfigurationListParams{})
	if err != nil {
		t.Fatalf("list configurations: %v", err)
	}
	if len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("configurations after delete = %+v, want only second", got)
	}
}

func createSantaConfigurationLabel(t *testing.T, db *database.DB, name string) int64 {
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
