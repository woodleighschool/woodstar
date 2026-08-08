//go:build postgres

package inventory

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestListForHostMissingHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	_, _, err := store.ListForHost(ctx, 999999, HostSoftwareListParams{})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("ListForHost missing host error = %v, want ErrNotFound", err)
	}
}

func TestReplaceHostSoftwarePersistsAppVersionsAndMarksEmptySnapshotFresh(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "inventory-version-evidence-host"},
		OrbitNodeKey: "inventory-version-evidence-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	firstEntry := HostSoftwareEntry{
		Name:               "Version Evidence",
		Version:            "1.2.3",
		Source:             "apps",
		BundleIdentifier:   "com.example.version-evidence",
		BundleShortVersion: "1.2.3",
		BundleVersion:      "100",
		InstalledPath:      "/Applications/Version Evidence.app",
	}
	secondEntry := firstEntry
	secondEntry.BundleVersion = "200"
	secondEntry.InstalledPath = "/Applications/Utilities/Version Evidence.app"
	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{firstEntry, secondEntry}); err != nil {
		t.Fatalf("replace initial software: %v", err)
	}

	rows, err := db.Pool().Query(ctx, `
SELECT installed_path, bundle_short_version, bundle_version
FROM host_software_installed_paths
WHERE host_id = $1
ORDER BY installed_path`, host.ID)
	if err != nil {
		t.Fatalf("read persisted app versions: %v", err)
	}
	type pathVersion struct {
		Path         string
		ShortVersion string
		BuildVersion string
	}
	versions, err := pgx.CollectRows(rows, pgx.RowToStructByPos[pathVersion])
	if err != nil {
		t.Fatalf("collect persisted app versions: %v", err)
	}
	if len(versions) != 2 ||
		versions[0] != (pathVersion{"/Applications/Utilities/Version Evidence.app", "1.2.3", "200"}) ||
		versions[1] != (pathVersion{"/Applications/Version Evidence.app", "1.2.3", "100"}) {
		t.Fatalf("persisted path versions = %#v, want versions associated with each path", versions)
	}

	firstEntry.BundleVersion = "101"
	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{firstEntry, secondEntry}); err != nil {
		t.Fatalf("replace updated software: %v", err)
	}
	var shortVersion, buildVersion string
	if err := db.Pool().QueryRow(ctx, `
SELECT bundle_short_version, bundle_version
FROM host_software_installed_paths
WHERE host_id = $1 AND installed_path = $2`,
		host.ID,
		firstEntry.InstalledPath,
	).Scan(&shortVersion, &buildVersion); err != nil {
		t.Fatalf("read updated app versions: %v", err)
	}
	if shortVersion != "1.2.3" || buildVersion != "101" {
		t.Fatalf("updated app versions = %q/%q, want 1.2.3/101", shortVersion, buildVersion)
	}

	oldFreshness := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	if _, err := db.Pool().Exec(ctx,
		`UPDATE hosts SET software_inventory_updated_at = $2 WHERE id = $1`,
		host.ID, oldFreshness,
	); err != nil {
		t.Fatalf("set old software inventory freshness: %v", err)
	}
	if err := store.ReplaceHostSoftware(ctx, host.ID, nil); err != nil {
		t.Fatalf("replace with successful empty snapshot: %v", err)
	}

	var updatedAt time.Time
	var installedCount int
	if err := db.Pool().QueryRow(ctx, `
SELECT software_inventory_updated_at,
       (SELECT count(*) FROM host_software WHERE host_id = $1)
FROM hosts
WHERE id = $1`, host.ID).Scan(&updatedAt, &installedCount); err != nil {
		t.Fatalf("read empty snapshot evidence: %v", err)
	}
	if !updatedAt.After(oldFreshness) || installedCount != 0 {
		t.Fatalf("empty snapshot evidence = %v with %d rows, want newer timestamp and no rows", updatedAt, installedCount)
	}
}

func TestListForHostEmptyHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "inventory-empty-host"},
		OrbitNodeKey: "inventory-empty-host-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	rows, count, err := store.ListForHost(ctx, host.ID, HostSoftwareListParams{})
	if err != nil {
		t.Fatalf("ListForHost empty host: %v", err)
	}
	if len(rows) != 0 || count != 0 {
		t.Fatalf("ListForHost empty host = %d rows count %d, want empty page", len(rows), count)
	}
}
