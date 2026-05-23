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

func TestConfigurationStoreValidatesConflictsAndReplacesEditableShape(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := configurations.NewStore(db)
	firstLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration First")
	secondLabelID := createSantaConfigurationLabel(t, db, "Santa Configuration Second")

	_, err := store.CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name:                    "short sync",
		FullSyncIntervalSeconds: new(59),
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("short full sync error = %v, want ErrInvalidInput", err)
	}

	_, err = store.CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name: "remount without flags",
		RemovableMediaPolicy: &configurations.RemovableMediaPolicy{
			Action: configurations.RemovableMediaActionRemount,
		},
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("remount without flags error = %v, want ErrInvalidInput", err)
	}

	enableBundles := true
	fullSyncInterval := 120
	config, err := store.CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name:                    " Baseline ",
		ClientMode:              configurations.ClientModeLockdown,
		EnableBundles:           &enableBundles,
		FullSyncIntervalSeconds: &fullSyncInterval,
		RemovableMediaPolicy: &configurations.RemovableMediaPolicy{
			Action: configurations.RemovableMediaActionRemount,
			RemountFlags: []string{
				" rw ",
				"nosuid",
			},
		},
		LabelIDs: []int64{firstLabelID},
	})
	if err != nil {
		t.Fatalf("create configuration: %v", err)
	}
	if config.Name != "Baseline" || config.Position != 0 || config.ClientMode != configurations.ClientModeLockdown {
		t.Fatalf("configuration was not cleaned: %+v", config)
	}
	if config.EnableBundles == nil || !*config.EnableBundles ||
		config.RemovableMediaPolicy == nil ||
		len(config.RemovableMediaPolicy.RemountFlags) != 2 {
		t.Fatalf("settings were not preserved: %+v", config)
	}
	if len(config.LabelIDs) != 1 || config.LabelIDs[0] != firstLabelID {
		t.Fatalf("label IDs = %v, want [%d]", config.LabelIDs, firstLabelID)
	}

	_, err = store.CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name:     "Conflicting",
		LabelIDs: []int64{firstLabelID},
	})
	var conflict *configurations.ConfigurationLabelConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("label conflict error = %v, want ConfigurationLabelConflictError", err)
	}
	if conflict.LabelID != firstLabelID || conflict.ConfigurationID != config.ID ||
		conflict.ConfigurationName != "Baseline" {
		t.Fatalf("conflict = %+v, want existing configuration details", conflict)
	}

	updated, err := store.UpdateConfiguration(ctx, config.ID, configurations.ConfigurationMutation{
		Name:     " Updated ",
		LabelIDs: []int64{secondLabelID},
	})
	if err != nil {
		t.Fatalf("update configuration: %v", err)
	}
	if updated.Name != "Updated" || updated.ClientMode != configurations.ClientModeMonitor {
		t.Fatalf("updated configuration = %+v", updated)
	}
	if updated.EnableBundles != nil || updated.RemovableMediaPolicy != nil {
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

	first, err := store.CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name:     "First",
		LabelIDs: []int64{firstLabelID},
	})
	if err != nil {
		t.Fatalf("create first configuration: %v", err)
	}
	second, err := store.CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name:     "Second",
		LabelIDs: []int64{secondLabelID},
	})
	if err != nil {
		t.Fatalf("create second configuration: %v", err)
	}

	resolved, err := store.ResolveConfigurationForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve configuration: %v", err)
	}
	if resolved == nil || resolved.ID != first.ID || resolved.MatchedViaLabel == nil ||
		resolved.MatchedViaLabel.ID != firstLabelID {
		t.Fatalf("resolved configuration = %+v, want first configuration", resolved)
	}

	if err := store.ReorderConfigurations(ctx, []int64{second.ID, first.ID}); err != nil {
		t.Fatalf("reorder configurations: %v", err)
	}
	resolved, err = store.ResolveConfigurationForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve configuration after reorder: %v", err)
	}
	if resolved == nil || resolved.ID != second.ID || resolved.MatchedViaLabel == nil ||
		resolved.MatchedViaLabel.ID != secondLabelID {
		t.Fatalf("resolved configuration after reorder = %+v, want second configuration", resolved)
	}

	err = store.ReorderConfigurations(ctx, []int64{second.ID})
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
