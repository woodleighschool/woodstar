//go:build postgres

package packages

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodstar/internal/fault"
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
		SoftwareID:        softwareID,
		Version:           "1.0.0",
		InstallerType:     InstallerTypePkg,
		InstallerObjectID: &oldInstaller.ID,
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
	if _, err := registry.GetByID(ctx, oldInstaller.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("get replaced installer error = %v, want ErrNotFound", err)
	}
	if _, err := registry.GetByID(ctx, replacement.ID); err != nil {
		t.Fatalf("get replacement installer: %v", err)
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

func insertSoftware(t *testing.T, ctx context.Context, db *pgxpool.Pool, name string) int64 {
	t.Helper()
	var id int64
	err := db.
		QueryRow(ctx, `INSERT INTO munki_software (name, display_name) VALUES ($1, $1) RETURNING id`, name).
		Scan(&id)
	if err != nil {
		t.Fatalf("insert munki_software: %v", err)
	}
	return id
}
