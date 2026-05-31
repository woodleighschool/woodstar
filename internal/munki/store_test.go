package munki_test

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
)

func TestHostStatusUpsertAndDetail(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := munki.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-host-observation-uuid", Serial: "C02MUNKI"},
		OrbitNodeKey: "munki-host-observation-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	if detail, err := store.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load absent munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("absent munki detail = %+v, want nil", detail)
	}

	success := true
	if err := store.UpsertHostStatus(ctx, munki.HostStatusObservation{
		HostID:          host.ID,
		Version:         "7.1.2.5700",
		ManifestName:    "site_default",
		ConsoleUser:     "alice",
		Success:         &success,
		Errors:          []string{"first error"},
		Warnings:        []string{"first warning"},
		ProblemInstalls: []string{"Broken App"},
		RunStartedAt:    "2026-05-31 19:23:00 +1000",
		RunEndedAt:      "2026-05-31 19:24:14 +1000",
	}); err != nil {
		t.Fatalf("upsert munki host status: %v", err)
	}
	if err := store.ReplaceHostItems(ctx, host.ID, []munki.HostItem{
		{Name: "GoogleChrome", Installed: true, InstalledVersion: "148.0", RunEndedAt: "2026-05-31 19:24:14 +1000"},
		{Name: "Optional App", Installed: false},
	}); err != nil {
		t.Fatalf("replace munki host items: %v", err)
	}

	detail, err := store.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load munki detail: %v", err)
	}
	if detail == nil {
		t.Fatal("munki detail is nil")
	}
	if detail.Version != "7.1.2.5700" || detail.ManifestName != "site_default" {
		t.Fatalf("detail = %+v, want version and manifest", detail)
	}
	if detail.Success == nil || !*detail.Success {
		t.Fatalf("success = %v, want true", detail.Success)
	}
	if len(detail.Items) != 2 || detail.Items[0].Name != "GoogleChrome" || !detail.Items[0].Installed {
		t.Fatalf("items = %+v", detail.Items)
	}

	if err := store.ReplaceHostItems(
		ctx,
		host.ID,
		[]munki.HostItem{{Name: "Replacement", Installed: true}},
	); err != nil {
		t.Fatalf("replace munki host items again: %v", err)
	}
	detail, err = store.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load munki detail after replace: %v", err)
	}
	if len(detail.Items) != 1 || detail.Items[0].Name != "Replacement" {
		t.Fatalf("items after replace = %+v", detail.Items)
	}

	if err := store.ClearHostStatus(ctx, host.ID); err != nil {
		t.Fatalf("clear munki host status: %v", err)
	}
	if detail, err := store.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load cleared munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("cleared munki detail = %+v, want nil", detail)
	}
}
