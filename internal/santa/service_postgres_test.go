//go:build postgres

package santa_test

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestSyncServiceRuleDownloadUsesPreflightSnapshot(t *testing.T) {
	const (
		binaryIdentifier      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		certificateIdentifier = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	ruleStore := santarules.NewStore(db)
	configurationStore := configurations.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurationStore,
		Events:         santaevents.NewStore(db),
		Rules:          ruleStore,
		Sync:           syncstate.NewStore(db),
		Heartbeats:     heartbeats.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:   "santa-sync-host",
			Serial: "SANTASYNC",
		},
		OrbitNodeKey: "santa-sync-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	labelID := createSantaConfigurationLabel(t, db, "Santa Sync")
	if err := labelStore.SetMembership(ctx, labelID, host.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}
	configuration, err := configurationStore.Create(ctx, configurations.ConfigurationMutation{
		Name:                     "Santa Sync",
		ClientMode:               configurations.ClientModeMonitor,
		OverrideFileAccessAction: configurations.FileAccessActionNone,
		FullSyncIntervalSeconds:  600,
		BatchSize:                50,
		Targets: configurations.ConfigurationTargets{
			Include: []targeting.LabelRef{{LabelID: labelID}},
		},
	})
	if err != nil {
		t.Fatalf("create configuration: %v", err)
	}
	if _, err := ruleStore.Create(ctx, santarules.RuleMutation{
		ConfigurationID: configuration.ID,
		RuleType:        santarules.RuleTypeBinary,
		Identifier:      binaryIdentifier,
		Name:            "Blocked binary",
		Policy:          santarules.PolicyBlocklist,
		Targets: santarules.RuleTargets{
			Include: []targeting.LabelRef{{LabelID: labelID}},
		},
	}); err != nil {
		t.Fatalf("create initial rule: %v", err)
	}

	if _, err := service.Preflight(ctx, "santa-sync-host", heartbeats.Contact{}, santa.PreflightRequest{
		SerialNumber:     "SANTASYNC",
		RulesHash:        "00000000000000000000000000000000",
		RequestCleanSync: true,
	}); err != nil {
		t.Fatalf("freeze desired rules at preflight: %v", err)
	}
	if _, err := ruleStore.Create(ctx, santarules.RuleMutation{
		ConfigurationID: configuration.ID,
		RuleType:        santarules.RuleTypeCertificate,
		Identifier:      certificateIdentifier,
		Name:            "Blocked certificate",
		Policy:          santarules.PolicyBlocklist,
		Targets: santarules.RuleTargets{
			Include: []targeting.LabelRef{{LabelID: labelID}},
		},
	}); err != nil {
		t.Fatalf("create rule after preflight: %v", err)
	}

	frozenDownload, err := service.RuleDownload(ctx, "santa-sync-host", heartbeats.Contact{}, santa.RuleDownloadRequest{})
	if err != nil {
		t.Fatalf("download frozen rules: %v", err)
	}
	if len(frozenDownload.Rules) != 1 || frozenDownload.Rules[0].Identifier != binaryIdentifier {
		t.Fatalf("frozen download = %+v, want only the preflight snapshot", frozenDownload.Rules)
	}
}

func createSantaConfigurationLabel(t *testing.T, db *pgxpool.Pool, name string) int64 {
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
