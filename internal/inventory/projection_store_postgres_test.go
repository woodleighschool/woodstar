//go:build postgres

package inventory

import (
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestReplaceHostSoftwareReconcilesSnapshotAsSet(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "software-snapshot-set"},
		OrbitNodeKey: "software-snapshot-set-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	firstOpenedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	lastOpenedAt := firstOpenedAt.Add(time.Hour)
	app := HostSoftwareEntry{
		Name:             "Snapshot App",
		Version:          "1.0",
		Source:           "apps",
		BundleIdentifier: "com.example.snapshot",
		Vendor:           "Example",
		InstalledPath:    "/Applications/Snapshot App.app",
		Signature: &SoftwareCodeSignature{
			Signed:     true,
			Identifier: "first-observation",
		},
		LastOpenedAt: &firstOpenedAt,
	}
	updatedApp := app
	updatedApp.Signature = &SoftwareCodeSignature{
		Signed:     true,
		Identifier: "last-observation",
	}
	updatedApp.LastOpenedAt = &lastOpenedAt
	command := HostSoftwareEntry{
		Name:          "Snapshot CLI",
		Version:       "2.0",
		Source:        SoftwareSourceHomebrewPackages,
		Vendor:        "Example",
		InstalledPath: "/usr/local/bin/snapshot",
	}

	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{
		app,
		updatedApp,
		command,
	}); err != nil {
		t.Fatalf("replace software snapshot: %v", err)
	}

	var titleCount, softwareCount, hostSoftwareCount, pathCount int
	if err := db.QueryRow(ctx, `
SELECT
    (SELECT count(*) FROM software_titles),
    (SELECT count(*) FROM software),
    (SELECT count(*) FROM host_software WHERE host_id = $1),
    (SELECT count(*) FROM host_software_installed_paths WHERE host_id = $1)`, host.ID).
		Scan(&titleCount, &softwareCount, &hostSoftwareCount, &pathCount); err != nil {
		t.Fatalf("count software snapshot rows: %v", err)
	}
	if titleCount != 2 || softwareCount != 2 || hostSoftwareCount != 2 || pathCount != 2 {
		t.Fatalf(
			"snapshot counts = titles %d, software %d, host links %d, paths %d; want 2 each",
			titleCount, softwareCount, hostSoftwareCount, pathCount,
		)
	}

	var identifier string
	var storedLastOpenedAt time.Time
	if err := db.QueryRow(ctx, `
SELECT path.identifier, host_software.last_opened_at
FROM host_software
JOIN software ON software.id = host_software.software_id
JOIN host_software_installed_paths path
  ON path.host_id = host_software.host_id
 AND path.software_id = host_software.software_id
WHERE host_software.host_id = $1
  AND software.bundle_identifier = 'com.example.snapshot'`, host.ID).
		Scan(&identifier, &storedLastOpenedAt); err != nil {
		t.Fatalf("load duplicate observation result: %v", err)
	}
	if identifier != "last-observation" || !storedLastOpenedAt.Equal(lastOpenedAt) {
		t.Fatalf(
			"duplicate observation = identifier %q, last opened %v; want last observation at %v",
			identifier, storedLastOpenedAt, lastOpenedAt,
		)
	}

	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{command}); err != nil {
		t.Fatalf("replace with reduced snapshot: %v", err)
	}
	if err := db.QueryRow(ctx, `
SELECT
    (SELECT count(*) FROM host_software WHERE host_id = $1),
    (SELECT count(*) FROM host_software_installed_paths WHERE host_id = $1)`, host.ID).
		Scan(&hostSoftwareCount, &pathCount); err != nil {
		t.Fatalf("count reduced snapshot rows: %v", err)
	}
	if hostSoftwareCount != 1 || pathCount != 1 {
		t.Fatalf(
			"reduced snapshot counts = host links %d, paths %d; want 1 each",
			hostSoftwareCount, pathCount,
		)
	}
}

func TestReplaceHostSoftwareRejectsUnsupportedSourceWithoutReplacingSnapshot(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "inventory-source-host"},
		OrbitNodeKey: "inventory-source-host-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{{
		Name:    "Known App",
		Version: "1.0",
		Source:  SoftwareSourceApps,
	}}); err != nil {
		t.Fatalf("seed software: %v", err)
	}

	err = store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{{
		Name:    "Unknown App",
		Version: "1.0",
		Source:  SoftwareSource("unknown_source"),
	}})
	if err == nil {
		t.Fatal("ReplaceHostSoftware with unsupported source succeeded")
	}

	rows, count, err := store.ListForHost(ctx, host.ID, HostSoftwareListParams{})
	if err != nil {
		t.Fatalf("ListForHost after rejected snapshot: %v", err)
	}
	if count != 1 || len(rows) != 1 || rows[0].Name != "Known App" {
		t.Fatalf("software after rejected snapshot = (%d, %+v), want the prior snapshot", count, rows)
	}
}
