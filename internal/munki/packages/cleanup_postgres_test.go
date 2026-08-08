//go:build postgres

package packages

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestPackageUpdateSucceedsWhenReplacedInstallerBytesCannotBeRemoved(t *testing.T) {
	db, ctx := testdb.Open(t)
	requestCtx, cancelRequest := context.WithCancel(ctx)
	defer cancelRequest()
	registry := storage.NewObjectStore(
		db,
		unavailableBackend{cancelRequest: cancelRequest},
		slog.New(slog.DiscardHandler),
	)
	store := NewStore(db, registry)
	softwareID := insertSoftware(t, requestCtx, db, "CleanupFailure")
	oldInstaller := createAvailableInstaller(t, requestCtx, registry, "old.pkg")
	replacement := createAvailableInstaller(t, requestCtx, registry, "replacement.pkg")

	pkg, err := store.Create(requestCtx, PackageCreateMutation{
		SoftwareID: softwareID,
		PackageMutation: PackageMutation{
			Version:           "1.0.0",
			InstallerType:     InstallerTypePkg,
			InstallerObjectID: &oldInstaller.ID,
		},
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	updated, err := store.Update(requestCtx, pkg.ID, PackageMutation{
		Version:           pkg.Version,
		InstallerType:     InstallerTypePkg,
		InstallerObjectID: &replacement.ID,
	})
	if err != nil {
		t.Fatalf("update package: %v", err)
	}
	if updated.InstallerObjectID == nil || *updated.InstallerObjectID != replacement.ID {
		t.Fatalf("installer object = %v, want %d", updated.InstallerObjectID, replacement.ID)
	}
	if requestCtx.Err() == nil {
		t.Fatal("cleanup did not cancel the request context")
	}
	if _, err := registry.GetByID(ctx, oldInstaller.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get replaced installer error = %v, want ErrNotFound", err)
	}
	if _, err := registry.GetByID(ctx, replacement.ID); err != nil {
		t.Fatalf("get replacement installer: %v", err)
	}
}

func TestPackageDeleteReadsInstallerAfterConcurrentUpdateSerialization(t *testing.T) {
	db, ctx := testdb.Open(t)
	registry := storage.NewObjectStore(db, nil, slog.New(slog.DiscardHandler))
	store := NewStore(db, registry)
	softwareID := insertSoftware(t, ctx, db, "SerializedCleanup")
	oldInstaller := createAvailableInstaller(t, ctx, registry, "serialized-old.pkg")
	replacement := createAvailableInstaller(t, ctx, registry, "serialized-replacement.pkg")

	pkg, err := store.Create(ctx, PackageCreateMutation{
		SoftwareID: softwareID,
		PackageMutation: PackageMutation{
			Version:           "1.0.0",
			InstallerType:     InstallerTypePkg,
			InstallerObjectID: &oldInstaller.ID,
		},
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	tx, err := db.Pool().Begin(ctx)
	if err != nil {
		t.Fatalf("begin installer update: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.WithoutCancel(ctx)) })
	if _, err := tx.Exec(ctx, `SELECT id FROM munki_software WHERE id = $1 FOR UPDATE`, softwareID); err != nil {
		t.Fatalf("lock software: %v", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE munki_packages
SET installer_object_id = $2
WHERE id = $1`, pkg.ID, replacement.ID); err != nil {
		t.Fatalf("swap installer: %v", err)
	}

	type deleteResult struct {
		deleted int
		err     error
	}
	result := make(chan deleteResult, 1)
	go func() {
		deleted, err := store.DeleteMany(ctx, []int64{pkg.ID})
		result <- deleteResult{deleted: deleted, err: err}
	}()
	waitForDetectorSoftwareLock(t, db)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit installer update: %v", err)
	}

	select {
	case got := <-result:
		if got.err != nil || got.deleted != 1 {
			t.Fatalf("delete package = %d, %v, want one", got.deleted, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("delete package did not finish after installer update committed")
	}
	if _, err := registry.GetByID(ctx, replacement.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("replacement installer cleanup error = %v, want ErrNotFound", err)
	}
	if _, err := registry.GetByID(ctx, oldInstaller.ID); err != nil {
		t.Fatalf("old installer should not be stale cleanup target: %v", err)
	}
}

func waitForDetectorSoftwareLock(t *testing.T, db *database.DB) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		var waiting bool
		if err := db.Pool().QueryRow(t.Context(), `
SELECT EXISTS (
    SELECT 1
    FROM pg_stat_activity
    WHERE datname = current_database()
      AND wait_event_type = 'Lock'
      AND query LIKE '%FROM munki_software%FOR UPDATE%'
)`).Scan(&waiting); err != nil {
			t.Fatalf("inspect waiting software lock: %v", err)
		}
		if waiting {
			return
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			t.Fatal("DeleteMany did not wait on the software lock")
		}
	}
}

type unavailableBackend struct {
	cancelRequest context.CancelFunc
}

func (b unavailableBackend) Delete(context.Context, string) error {
	b.cancelRequest()
	return errors.New("backend unavailable")
}

func createAvailableInstaller(
	t *testing.T,
	ctx context.Context,
	registry *storage.ObjectStore,
	filename string,
) *storage.Object {
	t.Helper()
	object, err := registry.CreatePending(ctx, ObjectPrefix, filename)
	if err != nil {
		t.Fatalf("create pending installer: %v", err)
	}
	object, err = registry.MarkAvailable(
		ctx,
		object.ID,
		1,
		"application/octet-stream",
		strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatalf("finalize installer: %v", err)
	}
	return object
}

func insertSoftware(t *testing.T, ctx context.Context, db *database.DB, name string) int64 {
	t.Helper()
	var id int64
	err := db.Pool().
		QueryRow(ctx, `INSERT INTO munki_software (name, display_name) VALUES ($1, $1) RETURNING id`, name).
		Scan(&id)
	if err != nil {
		t.Fatalf("insert munki_software: %v", err)
	}
	return id
}
