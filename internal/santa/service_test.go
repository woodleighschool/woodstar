package santa_test

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func TestSyncServiceFreezesDownloadsAndPromotesCleanSnapshot(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := santa.NewStore(db)
	ruleStore := santarules.NewStore(db)
	configurationStore := configurations.NewStore(db)
	service := santa.NewService(santa.Dependencies{
		HostStore:      store,
		Configurations: configurationStore,
		Events:         santaevents.NewStore(db),
		Rules:          ruleStore,
		Sync:           syncstate.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID:   "santa-sync-host",
		HardwareSerial: "SANTASYNC",
		OrbitNodeKey:   "santa-sync-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	labelID := createSantaConfigurationLabel(t, db, "Santa Sync")
	if err := labelStore.SetMembership(ctx, labelID, host.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}
	if _, err := configurationStore.CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name:                    "Sync Config",
		ClientMode:              configurations.ClientModeLockdown,
		EnableBundles:           true,
		FullSyncIntervalSeconds: 120,
		BatchSize:               50,
		LabelIDs:                []int64{labelID},
	}); err != nil {
		t.Fatalf("create configuration: %v", err)
	}
	if _, err := ruleStore.CreateRule(ctx, santarules.RuleMutation{
		RuleType:      santarules.RuleTypeBinary,
		Identifier:    "binary-sha",
		CustomMessage: "Blocked",
		Includes: []santarules.RuleIncludeWrite{{
			Policy:   santarules.PolicyBlocklist,
			LabelIDs: []int64{labelID},
		}},
	}); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	preflight, err := service.Preflight(ctx, "santa-sync-host", santa.PreflightRequest{
		SerialNumber:     "SANTASYNC",
		Version:          "2026.2",
		ClientMode:       configurations.ClientModeMonitor,
		RequestCleanSync: true,
		RulesHash:        "opaque-client-hash",
	})
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if preflight.SyncType != syncstate.SyncTypeClean {
		t.Fatalf("sync type = %v, want clean", preflight.SyncType)
	}
	if preflight.Configuration == nil || preflight.Configuration.ClientMode != configurations.ClientModeLockdown {
		t.Fatalf("configuration = %+v, want lockdown", preflight.Configuration)
	}
	if !preflight.Configuration.EnableBundles {
		t.Fatalf("enable bundles = %v, want true", preflight.Configuration.EnableBundles)
	}
	if preflight.Configuration.FullSyncIntervalSeconds != 120 {
		t.Fatalf("full sync interval = %v, want 120", preflight.Configuration.FullSyncIntervalSeconds)
	}

	download, err := service.RuleDownload(ctx, "santa-sync-host", santa.RuleDownloadRequest{})
	if err != nil {
		t.Fatalf("rule download: %v", err)
	}
	if len(download.Rules) != 1 {
		t.Fatalf("downloaded rules = %+v, want one", download.Rules)
	}
	if download.Rules[0].Identifier != "binary-sha" ||
		download.Rules[0].Policy != string(santarules.PolicyBlocklist) ||
		download.Rules[0].RuleType != string(santarules.RuleTypeBinary) ||
		download.Rules[0].CustomMessage != "Blocked" {
		t.Fatalf("downloaded rule = %+v", download.Rules[0])
	}

	if _, err := ruleStore.CreateRule(ctx, santarules.RuleMutation{
		RuleType:   santarules.RuleTypeCertificate,
		Identifier: "cert-sha",
		Includes: []santarules.RuleIncludeWrite{{
			Policy:   santarules.PolicyBlocklist,
			LabelIDs: []int64{labelID},
		}},
	}); err != nil {
		t.Fatalf("create post-preflight rule: %v", err)
	}
	frozenDownload, err := service.RuleDownload(ctx, "santa-sync-host", santa.RuleDownloadRequest{})
	if err != nil {
		t.Fatalf("rule download after desired change: %v", err)
	}
	if len(frozenDownload.Rules) != 1 || frozenDownload.Rules[0].Identifier != "binary-sha" {
		t.Fatalf("frozen download = %+v, want original preflight payload", frozenDownload.Rules)
	}

	if _, err := service.Postflight(ctx, "santa-sync-host", santa.PostflightRequest{
		RulesReceived:  1,
		RulesProcessed: 1,
		RulesHash:      "new-client-hash",
	}); err != nil {
		t.Fatalf("postflight: %v", err)
	}

	hostState := santa.NewHostStateService(store, configurationStore)
	state, err := hostState.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load host state: %v", err)
	}
	if state.RuleSync.DesiredCount != 1 || state.RuleSync.AppliedCount != 1 || state.RuleSync.PendingCount != 0 {
		t.Fatalf("rule sync = %+v, want promoted clean snapshot", state.RuleSync)
	}
	if state.RuleSync.LastCleanSyncAt == nil {
		t.Fatalf("last clean sync was not recorded")
	}
}

func createSantaConfigurationLabel(t *testing.T, db *database.DB, name string) int64 {
	t.Helper()

	label, err := labels.NewStore(db).Create(t.Context(), labels.LabelCreate{
		Name:                name,
		LabelType:           labels.LabelTypeRegular,
		LabelMembershipType: labels.LabelMembershipTypeManual,
		Platforms: []scope.Platform{
			scope.PlatformDarwin,
			scope.PlatformWindows,
			scope.PlatformLinux,
		},
	})
	if err != nil {
		t.Fatalf("create label %q: %v", name, err)
	}
	return label.ID
}
