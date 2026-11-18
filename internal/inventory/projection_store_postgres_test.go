//go:build postgres

package inventory

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/heartbeats"
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
	}, heartbeats.Contact{})

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

	if err := store.ReplaceHostSoftware(ctx, host.ID, fullSoftwareSnapshot(
		app,
		updatedApp,
		command,
	)); err != nil {
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

	if err := store.ReplaceHostSoftware(ctx, host.ID, fullSoftwareSnapshot(command)); err != nil {
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
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := store.ReplaceHostSoftware(ctx, host.ID, fullSoftwareSnapshot(HostSoftwareEntry{
		Name:    "Known App",
		Version: "1.0",
		Source:  SoftwareSourceApps,
	})); err != nil {
		t.Fatalf("seed software: %v", err)
	}

	err = store.ReplaceHostSoftware(ctx, host.ID, fullSoftwareSnapshot(HostSoftwareEntry{
		Name:    "Unknown App",
		Version: "1.0",
		Source:  SoftwareSource("unknown_source"),
	}))
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

func fullSoftwareSnapshot(entries ...HostSoftwareEntry) SoftwareSnapshot {
	return SoftwareSnapshot{
		Sources:                  softwareSourceValues,
		Entries:                  entries,
		SignaturesReported:       true,
		ExecutableHashesReported: true,
	}
}

func TestSoftwareReplayLeavesRowsUntouched(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	host := enrollCleanupTestHost(t, ctx, hostStore, "inventory-replay")
	other := enrollCleanupTestHost(t, ctx, hostStore, "inventory-replay-other")
	app := cleanupTestSoftware("1.0")
	app.InstalledPath = "/Applications/Cleanup.app"
	tool := HostSoftwareEntry{Name: "tool", Version: "2", Vendor: "Tool Vendor", Source: SoftwareSourceNPMPackages}
	snapshot := fullSoftwareSnapshot(app, tool)
	if err := store.ReplaceHostSoftware(ctx, host.ID, snapshot); err != nil {
		t.Fatal(err)
	}
	revisions := func() []string {
		t.Helper()
		var rows []string
		if err := db.QueryRow(ctx, `
SELECT ARRAY(SELECT revision FROM (
    SELECT 'title' AS kind, id, xmin::text AS revision FROM software_titles UNION ALL
    SELECT 'version', id, xmin::text FROM software UNION ALL
    SELECT 'link', software_id, xmin::text FROM host_software WHERE host_id = $1 UNION ALL
    SELECT 'path', id, xmin::text FROM host_software_installed_paths WHERE host_id = $1
) revisions ORDER BY kind, id)`, host.ID).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	before := revisions()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT id FROM software_titles FOR SHARE; SELECT id FROM software FOR SHARE`); err != nil {
		t.Fatal(err)
	}
	reportCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for _, id := range []int64{host.ID, other.ID} {
		if err := store.ReplaceHostSoftware(reportCtx, id, snapshot); err != nil {
			t.Fatalf("report unchanged catalog for host %d: %v", id, err)
		}
	}
	if got := revisions(); !slices.Equal(before, got) {
		t.Fatalf("report changed row revisions from %v to %v", before, got)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}

	app.Vendor = "App Vendor"
	if err := store.ReplaceHostSoftware(ctx, host.ID, fullSoftwareSnapshot(app)); err != nil {
		t.Fatal(err)
	}
	var vendor string
	if err := db.QueryRow(ctx, `SELECT vendor FROM software_titles WHERE bundle_identifier = $1`, app.BundleIdentifier).Scan(&vendor); err != nil {
		t.Fatal(err)
	}
	if vendor != app.Vendor {
		t.Fatalf("title vendor = %q, want %q", vendor, app.Vendor)
	}
}

func TestSoftwarePartialSnapshotPreservesUnreportedObservations(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	host := enrollCleanupTestHost(t, ctx, hosts.NewStore(db, labels.NewStore(db)), "inventory-partial")
	app := cleanupTestSoftware("1.0")
	app.InstalledPath = "/Applications/Cleanup.app"
	app.Signature = &SoftwareCodeSignature{Signed: true, Identifier: "cleanup"}
	app.ExecutableSHA256 = "hash"
	app.ExecutablePath = "/Applications/Cleanup.app/Contents/MacOS/Cleanup"
	extension := HostSoftwareEntry{Name: "Extension", Source: SoftwareSourceVSCodeExtensions, InstalledPath: "/plugins/extension"}

	if err := store.ReplaceHostSoftware(ctx, host.ID, fullSoftwareSnapshot(app, extension)); err != nil {
		t.Fatal(err)
	}
	app.Signature, app.ExecutableSHA256, app.ExecutablePath = nil, "", ""
	snapshot := SoftwareSnapshot{Entries: []HostSoftwareEntry{app}, Sources: []SoftwareSource{SoftwareSourceApps}}
	check := func(wantCount int, wantSignature bool, wantHash string) {
		t.Helper()
		if err := store.ReplaceHostSoftware(ctx, host.ID, snapshot); err != nil {
			t.Fatal(err)
		}
		software, count, err := store.ListForHost(ctx, host.ID, HostSoftwareListParams{})
		if err != nil {
			t.Fatal(err)
		}
		if count != wantCount {
			t.Fatalf("host has %d titles, want %d", count, wantCount)
		}
		for _, title := range software {
			if title.Source != SoftwareSourceApps {
				continue
			}
			path := title.InstalledVersions[0].Paths[0]
			if (path.Signature != nil) != wantSignature || path.ExecutableSHA256 != wantHash {
				t.Fatalf("path = %+v, want signature present %t and hash %q", path, wantSignature, wantHash)
			}
			return
		}
		t.Fatal("missing app")
	}
	check(2, true, "hash")
	snapshot.Sources = append(snapshot.Sources, SoftwareSourceVSCodeExtensions)
	snapshot.SignaturesReported = true
	check(1, false, "hash")
	snapshot.SignaturesReported, snapshot.ExecutableHashesReported = false, true
	check(1, false, "")
}

func TestConcurrentHostSoftwareSnapshotsReplaceEachOther(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	host := enrollCleanupTestHost(t, ctx, hosts.NewStore(db, labels.NewStore(db)), "inventory-concurrent")
	// Pause the first snapshot after inserting a membership that the other
	// transaction cannot see yet.
	if _, err := db.Exec(ctx, `
CREATE FUNCTION pause_membership() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_advisory_xact_lock_shared(71234);
    RETURN NEW;
END $$;
CREATE TRIGGER pause_membership AFTER INSERT ON host_software FOR EACH ROW EXECUTE FUNCTION pause_membership()`); err != nil {
		t.Fatal(err)
	}
	gate, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = gate.Rollback(ctx) }()
	if _, err := gate.Exec(ctx, `SELECT pg_advisory_xact_lock(71234)`); err != nil {
		t.Fatal(err)
	}
	reportCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	results := make(chan error, 2)
	for i, name := range []string{"First", "Second"} {
		go func() {
			results <- store.ReplaceHostSoftware(reportCtx, host.ID, fullSoftwareSnapshot(
				HostSoftwareEntry{Name: name, Source: SoftwareSourceApps, InstalledPath: "/" + name},
			))
		}()
		waitForSoftwareWriters(t, reportCtx, db, i+1)
	}
	if err := gate.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	software, count, err := store.ListForHost(ctx, host.ID, HostSoftwareListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(software) != 1 || software[0].Name != "Second" {
		t.Fatalf("host software = %+v, want second snapshot", software)
	}
}

func waitForSoftwareWriters(t *testing.T, ctx context.Context, db *pgxpool.Pool, count int) {
	t.Helper()
	for {
		var waiting int
		if err := db.QueryRow(ctx, `
SELECT count(*) FROM pg_stat_activity
WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting); err != nil {
			t.Fatalf("wait for %d concurrent writers: %v", count, err)
		}
		if waiting >= count {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}
