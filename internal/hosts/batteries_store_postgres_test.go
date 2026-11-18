//go:build postgres

package hosts

import "testing"

func TestBatterySnapshotKeepsIdentityAcrossReadings(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	host := enrollTestHost(t, ctx, store, "battery-readings", nil)
	battery := HostBattery{SerialNumber: "pack", CycleCount: new(int32(65))}
	if err := store.ReplaceBatteries(ctx, host.ID, []HostBattery{battery}); err != nil {
		t.Fatal(err)
	}
	initial, err := store.ListBatteries(ctx, host.ID)
	if err != nil || len(initial) != 1 || initial[0].CycleCount == nil || *initial[0].CycleCount != 65 {
		t.Fatalf("initial batteries = %+v, %v", initial, err)
	}
	reading := HostBattery{SerialNumber: "pack", PercentRemaining: new(88.5)}
	if err := store.ReplaceBatteries(ctx, host.ID, []HostBattery{battery, {}, reading}); err != nil {
		t.Fatal(err)
	}
	current, err := store.ListBatteries(ctx, host.ID)
	if err != nil || len(current) != 1 {
		t.Fatalf("current batteries = %+v, %v", current, err)
	}
	got := current[0]
	if got.ID != initial[0].ID || got.CycleCount != nil || got.PercentRemaining == nil || *got.PercentRemaining != 88.5 {
		t.Fatalf("current battery = %+v", got)
	}
	if err := store.ReplaceBatteries(ctx, host.ID, nil); err != nil {
		t.Fatal(err)
	}
	if got, err := store.ListBatteries(ctx, host.ID); err != nil || len(got) != 0 {
		t.Fatalf("empty snapshot = %+v, %v", got, err)
	}
}
