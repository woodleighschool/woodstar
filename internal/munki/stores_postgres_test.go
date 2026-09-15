//go:build postgres

package munki_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/bloby"

	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/testutil/testbloby"
)

type munkiStores struct {
	db        *pgxpool.Pool
	objects   *bloby.Service
	hoststate *munki.Store
	packages  *packages.Store
	software  *munkisoftware.Store
}

func newMunkiStores(t *testing.T, db *pgxpool.Pool) munkiStores {
	t.Helper()
	objectStore := testbloby.New(t, db)
	packageStore := packages.NewStore(db, objectStore)
	softwareStore := munkisoftware.NewStore(db, objectStore, packageStore)
	return munkiStores{
		db:        db,
		objects:   objectStore,
		hoststate: munki.NewStore(db),
		packages:  packageStore,
		software:  softwareStore,
	}
}

func createMunkiStorageObject(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	prefix, filename, contentSeed string,
) *bloby.Object {
	t.Helper()
	contentType := "application/octet-stream"
	if prefix == munkisoftware.IconObjectPrefix {
		contentType = "image/png"
	}
	object, err := stores.objects.Write(ctx, prefix, filename, contentType, []byte(strings.Repeat(contentSeed, 512)))
	if err != nil {
		t.Fatalf("write object: %v", err)
	}
	return object
}

func requireErrorIs(t *testing.T, operation string, err error, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("%s error = %v, want %v", operation, err, target)
	}
}

func allHostsLabelID(t *testing.T, ctx context.Context, labelStore *labels.Store) int64 {
	t.Helper()
	rows, _, err := labelStore.List(ctx, labels.LabelListParams{})
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}
	for _, row := range rows {
		if row.BuiltinKey != nil && *row.BuiltinKey == labels.BuiltinKeyAllHosts {
			return row.ID
		}
	}
	t.Fatalf("All Hosts label not found")
	return 0
}

func replaceTargets(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	title *munkisoftware.Software,
	includes []munkisoftware.Include,
) munkisoftware.Targets {
	t.Helper()
	if _, err := stores.software.Update(
		ctx,
		title.ID,
		softwareTargetMutation(title, includes, nil),
	); err != nil {
		t.Fatalf("replace software targets: %v", err)
	}
	targets, err := stores.software.TargetsForSoftware(ctx, title.ID)
	if err != nil {
		t.Fatalf("list software targets: %v", err)
	}
	return targets
}

func softwareTargetMutation(
	title *munkisoftware.Software,
	includes []munkisoftware.Include,
	excludeLabelIDs []int64,
) munkisoftware.UpdateMutation {
	displayName := ""
	if title.DisplayName != nil {
		displayName = *title.DisplayName
	}
	return munkisoftware.UpdateMutation{
		DisplayName:  displayName,
		Description:  title.Description,
		Category:     title.Category,
		Developer:    title.Developer,
		IconObjectID: title.IconObjectID,
		Targets: munkisoftware.Targets{
			Include: includes,
			Exclude: labelRefs(excludeLabelIDs),
		},
	}
}

func includeTarget(
	labelID int64,
	actions ...munkisoftware.Action,
) munkisoftware.Include {
	return munkisoftware.Include{
		LabelID: labelID,
		Package: munkisoftware.PackageSelector{
			Strategy: munkisoftware.PackageLatest,
		},
		Actions: actions,
	}
}

func includeSpecificTarget(
	labelID int64,
	action munkisoftware.Action,
	pinnedPackageID int64,
) munkisoftware.Include {
	return munkisoftware.Include{
		LabelID: labelID,
		Package: munkisoftware.PackageSelector{
			Strategy:  munkisoftware.PackageSpecific,
			PackageID: &pinnedPackageID,
		},
		Actions: []munkisoftware.Action{action},
	}
}

func labelRefs(ids []int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(ids))
	for i, id := range ids {
		refs[i] = targeting.LabelRef{LabelID: id}
	}
	return refs
}

func createMunkiPackage(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	softwareID int64,
	name string,
	version string,
) packages.Package {
	t.Helper()
	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: softwareID,
			Version:       version,
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true},
	)
	if err != nil {
		t.Fatalf("create pkg %s %s: %v", name, version, err)
	}
	return *pkg
}

func createMunkiPackageObject(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	location string,
	contentSeed string,
) *bloby.Object {
	t.Helper()
	return createMunkiStorageObject(t, ctx, stores, "munki/packages", location, contentSeed)
}

func assertBlockingApplications(t *testing.T, pkg packages.Package, wantNone bool, want []string) {
	t.Helper()
	if pkg.BlockingApplicationsNone != wantNone {
		t.Fatalf(
			"package %s blocking_applications_none = %t, want %t",
			pkg.Version,
			pkg.BlockingApplicationsNone,
			wantNone,
		)
	}
	if len(pkg.BlockingApplications) != len(want) {
		t.Fatalf("package %s blocking applications = %#v, want %#v", pkg.Version, pkg.BlockingApplications, want)
	}
	for i := range want {
		if pkg.BlockingApplications[i] != want[i] {
			t.Fatalf("package %s blocking applications = %#v, want %#v", pkg.Version, pkg.BlockingApplications, want)
		}
	}
}

func assertNoMunkiChildren(t *testing.T, ctx context.Context, stores munkiStores, softwareID int64) {
	t.Helper()
	pkgRows, pkgCount, err := stores.packages.List(ctx, packages.PackageListParams{SoftwareID: softwareID})
	if err != nil {
		t.Fatalf("list packages after delete: %v", err)
	}
	if pkgCount != 0 || len(pkgRows) != 0 {
		t.Fatalf("packages after delete = %+v count = %d, want none", pkgRows, pkgCount)
	}
	targets, err := stores.software.TargetsForSoftware(ctx, softwareID)
	if err != nil {
		t.Fatalf("list targets after delete: %v", err)
	}
	if len(targets.Include) != 0 || len(targets.Exclude) != 0 {
		t.Fatalf("targets after delete = %+v, want none", targets)
	}
}

func assertObjectDeleted(
	t *testing.T,
	ctx context.Context,
	objects *bloby.Service,
	objectID int64,
) {
	t.Helper()
	if _, err := objects.GetByID(ctx, objectID); !errors.Is(err, bloby.ErrNotFound) {
		t.Fatalf("get deleted object %d error = %v, want ErrNotFound", objectID, err)
	}
}

func createMunkiIconObject(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	location string,
	contentSeed string,
) *bloby.Object {
	t.Helper()
	return createMunkiStorageObject(t, ctx, stores, "munki/icons", location, contentSeed)
}

func runBlockedPatch(t *testing.T, db *pgxpool.Pool, table string, id int64, patch func(context.Context) error, mutate func(context.Context, pgx.Tx)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := tx.QueryRow(ctx, "SELECT id FROM "+table+" WHERE id = $1 FOR UPDATE", id).Scan(&id); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- patch(ctx) }()
	waitForBlockedPatch(t, ctx, db, tx.Conn().PgConn().PID(), done)
	mutate(ctx, tx)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func waitForBlockedPatch(t *testing.T, ctx context.Context, db *pgxpool.Pool, blocker uint32, done <-chan error) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname = current_database() AND $1::integer = ANY(pg_blocking_pids(pid)))`, blocker).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			return
		}
		select {
		case err := <-done:
			t.Fatalf("patch completed before acquiring the owner lock: %v", err)
		case <-ctx.Done():
			t.Fatal("patch did not wait for the owner lock")
		case <-ticker.C:
		}
	}
}

func readPatch[T any](t *testing.T, data string) T {
	t.Helper()
	var document T
	if err := json.Unmarshal([]byte(data), &document); err != nil {
		t.Fatal(err)
	}
	return document
}
