package santa_test

import (
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
)

func TestHostObservationUpsertAndDetail(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := santa.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID:   "santa-host-observation-uuid",
		HardwareSerial: "C02SANTA",
		OrbitNodeKey:   "santa-host-observation-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	if detail, err := store.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load absent santa detail: %v", err)
	} else if detail != nil {
		t.Fatalf("absent santa detail = %+v, want nil", detail)
	}

	seenAt := time.Date(2026, 5, 23, 10, 30, 0, 0, time.UTC)
	sipStatus := int16(1)
	if err := store.UpsertHostObservation(ctx, santa.HostObservation{
		HostID:             host.ID,
		MachineID:          "machine-uuid",
		SerialNumber:       "C02SANTA",
		Version:            "2026.1",
		ClientModeReported: santa.ClientModeMonitor,
		PrimaryUser:        "alice",
		PrimaryUserGroups:  []string{"staff", "admin"},
		SIPStatus:          &sipStatus,
		OSBuild:            "25A1",
		ModelIdentifier:    "Mac16,1",
		LastSeenAt:         &seenAt,
	}); err != nil {
		t.Fatalf("upsert santa host observation: %v", err)
	}

	detail, err := store.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load santa detail: %v", err)
	}
	if detail == nil {
		t.Fatal("santa detail is nil")
	}
	if !detail.Enrolled {
		t.Fatal("santa detail is not marked enrolled")
	}
	if detail.Version != "2026.1" {
		t.Fatalf("version = %q, want 2026.1", detail.Version)
	}
	if detail.ClientModeReported != santa.ClientModeMonitor {
		t.Fatalf("client mode = %q, want monitor", detail.ClientModeReported)
	}
	if detail.LastSyncAt == nil || !detail.LastSyncAt.Equal(seenAt) {
		t.Fatalf("last sync = %v, want %v", detail.LastSyncAt, seenAt)
	}
	if detail.RuleSync.DesiredCount != 0 || detail.RuleSync.AppliedCount != 0 || detail.RuleSync.PendingCount != 0 {
		t.Fatalf("rule sync = %+v, want zero counts", detail.RuleSync)
	}
}
