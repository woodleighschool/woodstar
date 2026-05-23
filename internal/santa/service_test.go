package santa_test

import (
	"testing"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa"
)

func TestSyncServiceFreezesDownloadsAndPromotesCleanSnapshot(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := santa.NewStore(db)
	service := santa.NewService(store)

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
	enableBundles := true
	fullSyncInterval := 120
	if _, err := store.CreateConfiguration(ctx, santa.ConfigurationCreate{
		Name:                    "Sync Config",
		ClientMode:              santa.ClientModeLockdown,
		EnableBundles:           &enableBundles,
		FullSyncIntervalSeconds: &fullSyncInterval,
		LabelIDs:                []int64{labelID},
	}); err != nil {
		t.Fatalf("create configuration: %v", err)
	}
	if _, err := store.CreateRule(ctx, santa.RuleCreate{
		RuleType:      santa.RuleTypeBinary,
		Identifier:    "binary-sha",
		CustomMessage: "Blocked",
		Includes: []santa.RuleIncludeWrite{{
			Policy:   santa.PolicyBlocklist,
			LabelIDs: []int64{labelID},
		}},
	}); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	preflight, err := service.HandlePreflight(ctx, "santa-sync-host", &syncv1.PreflightRequest{
		MachineId:        "santa-sync-host",
		SerialNumber:     "SANTASYNC",
		SantaVersion:     "2026.2",
		ClientMode:       syncv1.ClientMode_MONITOR,
		RequestCleanSync: true,
		RulesHash:        "opaque-client-hash",
	})
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if preflight.GetSyncType() != syncv1.SyncType_CLEAN {
		t.Fatalf("sync type = %v, want CLEAN", preflight.GetSyncType())
	}
	if preflight.GetClientMode() != syncv1.ClientMode_LOCKDOWN {
		t.Fatalf("client mode = %v, want LOCKDOWN", preflight.GetClientMode())
	}
	if preflight.EnableBundles == nil || !preflight.GetEnableBundles() {
		t.Fatalf("enable bundles = %v, want true", preflight.EnableBundles)
	}
	if preflight.GetFullSyncIntervalSeconds() != 120 {
		t.Fatalf("full sync interval = %d, want 120", preflight.GetFullSyncIntervalSeconds())
	}

	download, err := service.HandleRuleDownload(ctx, "santa-sync-host", &syncv1.RuleDownloadRequest{
		MachineId: "santa-sync-host",
	})
	if err != nil {
		t.Fatalf("rule download: %v", err)
	}
	if len(download.Rules) != 1 {
		t.Fatalf("downloaded rules = %+v, want one", download.Rules)
	}
	if download.Rules[0].GetIdentifier() != "binary-sha" ||
		download.Rules[0].GetPolicy() != syncv1.Policy_BLOCKLIST ||
		download.Rules[0].GetRuleType() != syncv1.RuleType_BINARY ||
		download.Rules[0].GetCustomMsg() != "Blocked" {
		t.Fatalf("downloaded rule = %+v", download.Rules[0])
	}

	if _, err := service.HandlePostflight(ctx, "santa-sync-host", &syncv1.PostflightRequest{
		MachineId:      "santa-sync-host",
		RulesReceived:  1,
		RulesProcessed: 1,
		SyncType:       syncv1.SyncType_CLEAN,
		RulesHash:      "new-client-hash",
	}); err != nil {
		t.Fatalf("postflight: %v", err)
	}

	state, err := store.LoadHostState(ctx, host.ID)
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
