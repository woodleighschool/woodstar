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
)

func TestSyncServiceFreezesDownloadsAndPromotesCleanSnapshot(t *testing.T) {
	const (
		binaryIdentifier      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		certificateIdentifier = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	ruleStore := santarules.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         santaevents.NewStore(db),
		Rules:          ruleStore,
		Sync:           syncstate.NewStore(db),
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
	if _, err := ruleStore.Create(ctx, santarules.RuleMutation{
		RuleType:   santarules.RuleTypeBinary,
		Identifier: binaryIdentifier,
		Name:       "Blocked binary",
		Targets: santarules.RuleTargets{
			Include: []santarules.RuleInclude{{
				Policy:  santarules.PolicyBlocklist,
				LabelID: labelID,
			}},
		},
	}); err != nil {
		t.Fatalf("create initial rule: %v", err)
	}

	if _, err := service.Preflight(ctx, "santa-sync-host", santa.PreflightRequest{
		SerialNumber:     "SANTASYNC",
		RulesHash:        "00000000000000000000000000000000",
		RequestCleanSync: true,
	}); err != nil {
		t.Fatalf("freeze desired rules at preflight: %v", err)
	}
	if _, err := ruleStore.Create(ctx, santarules.RuleMutation{
		RuleType:   santarules.RuleTypeCertificate,
		Identifier: certificateIdentifier,
		Name:       "Blocked certificate",
		Targets: santarules.RuleTargets{
			Include: []santarules.RuleInclude{{
				Policy:  santarules.PolicyBlocklist,
				LabelID: labelID,
			}},
		},
	}); err != nil {
		t.Fatalf("create rule after preflight: %v", err)
	}

	frozenDownload, err := service.RuleDownload(ctx, "santa-sync-host", santa.RuleDownloadRequest{})
	if err != nil {
		t.Fatalf("download frozen rules: %v", err)
	}
	if len(frozenDownload.Rules) != 1 || frozenDownload.Rules[0].Identifier != binaryIdentifier {
		t.Fatalf("frozen download = %+v, want only the preflight snapshot", frozenDownload.Rules)
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
