//go:build postgres

package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestPruneUnreferencedSoftwarePreservesReferencedVersions(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	hostOne := enrollCleanupTestHost(t, ctx, hostStore, "cleanup-shared-one")
	hostTwo := enrollCleanupTestHost(t, ctx, hostStore, "cleanup-shared-two")

	v1 := cleanupTestSoftware("1.0")
	if err := store.ReplaceHostSoftware(ctx, hostOne.ID, []HostSoftwareEntry{v1}); err != nil {
		t.Fatalf("report first host software: %v", err)
	}
	if err := store.ReplaceHostSoftware(ctx, hostTwo.ID, []HostSoftwareEntry{v1}); err != nil {
		t.Fatalf("report second host software: %v", err)
	}
	var titleID int64
	if err := db.QueryRow(ctx, `
SELECT id
FROM software_titles
WHERE bundle_identifier = 'com.example.cleanup'`).Scan(&titleID); err != nil {
		t.Fatalf("load software title ID: %v", err)
	}

	if err := store.ReplaceHostSoftware(ctx, hostOne.ID, nil); err != nil {
		t.Fatalf("clear first host software: %v", err)
	}
	result, err := store.PruneUnreferencedSoftware(ctx)
	if err != nil {
		t.Fatalf("prune shared software: %v", err)
	}
	if result != (CleanupResult{}) {
		t.Fatalf("shared software cleanup = %+v, want no deletions", result)
	}

	v2 := cleanupTestSoftware("2.0")
	if err := store.ReplaceHostSoftware(ctx, hostTwo.ID, []HostSoftwareEntry{v2}); err != nil {
		t.Fatalf("replace second host software version: %v", err)
	}
	result, err = store.PruneUnreferencedSoftware(ctx)
	if err != nil {
		t.Fatalf("prune old software version: %v", err)
	}
	if result.SoftwareVersions != 1 || result.SoftwareTitles != 0 {
		t.Fatalf("old version cleanup = %+v, want one version and no title", result)
	}
	title, err := store.GetTitle(ctx, titleID)
	if err != nil {
		t.Fatalf("load retained software title: %v", err)
	}
	if title.Versions.Count != 1 || len(title.Versions.Items) != 1 || title.Versions.Items[0].Version != "2.0" {
		t.Fatalf("retained versions = %+v, want version 2.0", title.Versions)
	}

	if err := store.ReplaceHostSoftware(ctx, hostTwo.ID, nil); err != nil {
		t.Fatalf("clear final host software: %v", err)
	}
	result, err = store.PruneUnreferencedSoftware(ctx)
	if err != nil {
		t.Fatalf("prune final software reference: %v", err)
	}
	if result.SoftwareVersions != 1 || result.SoftwareTitles != 1 {
		t.Fatalf("final cleanup = %+v, want one version and one title", result)
	}
	if _, err := store.GetTitle(ctx, titleID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("load pruned title error = %v, want ErrNotFound", err)
	}
	result, err = store.PruneUnreferencedSoftware(ctx)
	if err != nil {
		t.Fatalf("repeat cleanup: %v", err)
	}
	if result != (CleanupResult{}) {
		t.Fatalf("repeated cleanup = %+v, want no deletions", result)
	}
}

func TestPruneUnreferencedSoftwareAllowsFreshObservation(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	host := enrollCleanupTestHost(t, ctx, hostStore, "cleanup-reobserved")
	entry := cleanupTestSoftware("1.0")

	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{entry}); err != nil {
		t.Fatalf("report initial software: %v", err)
	}
	oldTitleID, oldSoftwareID := cleanupTestSoftwareIDs(t, ctx, db)
	if err := store.ReplaceHostSoftware(ctx, host.ID, nil); err != nil {
		t.Fatalf("clear initial software: %v", err)
	}
	if _, err := store.PruneUnreferencedSoftware(ctx); err != nil {
		t.Fatalf("prune initial software: %v", err)
	}

	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{entry}); err != nil {
		t.Fatalf("report software again: %v", err)
	}
	newTitleID, newSoftwareID := cleanupTestSoftwareIDs(t, ctx, db)
	if newTitleID == oldTitleID || newSoftwareID == oldSoftwareID {
		t.Fatalf(
			"re-observed IDs = title %d software %d, want fresh IDs after title %d software %d",
			newTitleID, newSoftwareID, oldTitleID, oldSoftwareID,
		)
	}
	var links int
	if err := db.QueryRow(ctx, `
SELECT count(*)
FROM host_software
WHERE host_id = $1 AND software_id = $2`, host.ID, newSoftwareID).Scan(&links); err != nil {
		t.Fatalf("count re-observed host link: %v", err)
	}
	if links != 1 {
		t.Fatalf("re-observed host links = %d, want 1", links)
	}
}

func TestPruneUnreferencedSoftwareAfterHostDeletion(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	host := enrollCleanupTestHost(t, ctx, hostStore, "cleanup-deleted-host")
	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{cleanupTestSoftware("1.0")}); err != nil {
		t.Fatalf("report software: %v", err)
	}
	titleID, _ := cleanupTestSoftwareIDs(t, ctx, db)
	if err := hostStore.Delete(ctx, host.ID); err != nil {
		t.Fatalf("delete host: %v", err)
	}

	result, err := store.PruneUnreferencedSoftware(ctx)
	if err != nil {
		t.Fatalf("prune deleted host software: %v", err)
	}
	if result.SoftwareVersions != 1 || result.SoftwareTitles != 1 {
		t.Fatalf("deleted host cleanup = %+v, want one version and one title", result)
	}
	if _, err := store.GetTitle(ctx, titleID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("load deleted host title error = %v, want ErrNotFound", err)
	}
}

func TestPruneUnreferencedSoftwareSkipsLockedInventory(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	host := enrollCleanupTestHost(t, ctx, hostStore, "cleanup-concurrent")
	entry := cleanupTestSoftware("1.0")
	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{entry}); err != nil {
		t.Fatalf("report software: %v", err)
	}
	_, softwareID := cleanupTestSoftwareIDs(t, ctx, db)
	if err := store.ReplaceHostSoftware(ctx, host.ID, nil); err != nil {
		t.Fatalf("clear software: %v", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin inventory transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE software SET updated_at = now() WHERE id = $1`, softwareID); err != nil {
		t.Fatalf("lock software row: %v", err)
	}
	cleanupCtx, cancel := context.WithTimeout(ctx, time.Second)
	result, err := store.PruneUnreferencedSoftware(cleanupCtx)
	cancel()
	if err != nil {
		t.Fatalf("prune around locked inventory: %v", err)
	}
	if result != (CleanupResult{}) {
		t.Fatalf("locked inventory cleanup = %+v, want no deletions", result)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO host_software (host_id, software_id)
VALUES ($1, $2)`, host.ID, softwareID); err != nil {
		t.Fatalf("link locked software: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit inventory transaction: %v", err)
	}

	result, err = store.PruneUnreferencedSoftware(ctx)
	if err != nil {
		t.Fatalf("prune linked inventory: %v", err)
	}
	if result != (CleanupResult{}) {
		t.Fatalf("linked inventory cleanup = %+v, want no deletions", result)
	}
}

func enrollCleanupTestHost(
	t *testing.T,
	ctx context.Context,
	store *hosts.Store,
	name string,
) *hosts.Host {
	t.Helper()
	host, err := store.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: name},
		OrbitNodeKey: name + "-orbit",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host %q: %v", name, err)
	}
	return host
}

func cleanupTestSoftware(version string) HostSoftwareEntry {
	return HostSoftwareEntry{
		Name:             "Cleanup App",
		Version:          version,
		Source:           "apps",
		BundleIdentifier: "com.example.cleanup",
	}
}

type cleanupTestQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func cleanupTestSoftwareIDs(t *testing.T, ctx context.Context, queryer cleanupTestQueryer) (int64, int64) {
	t.Helper()
	var titleID int64
	var softwareID int64
	if err := queryer.QueryRow(ctx, `
SELECT software_titles.id, software.id
FROM software_titles
JOIN software ON software.title_id = software_titles.id
WHERE software_titles.bundle_identifier = 'com.example.cleanup'`).Scan(&titleID, &softwareID); err != nil {
		t.Fatalf("load cleanup software IDs: %v", err)
	}
	return titleID, softwareID
}
