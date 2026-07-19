package mdp_test

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// newStore returns a store and the presence set the hub would normally write, so
// tests can mark a point online without standing up a live connection.
func newStore(db *database.DB) (*mdp.Store, *mdp.Presence) {
	store := mdp.NewStore(db, storage.NewObjectStore(db, nil), discardLogger())
	return store, store.Presence()
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func pointMutation(name string, cidrs []string) mdp.DistributionPointMutation {
	return mdp.DistributionPointMutation{
		Name:          name,
		Enabled:       true,
		ClientCIDRs:   cidrs,
		ClientBaseURL: "https://mdp.example",
	}
}

// seedAvailablePackage inserts a software item, an available installer object,
// and the package that links them, returning the package id.
func seedAvailablePackage(
	t *testing.T,
	db *database.DB,
	ctx context.Context,
	name string,
	sha256 string,
	size int64,
) int64 {
	t.Helper()
	var softwareID int64
	if err := db.Pool().QueryRow(ctx,
		`INSERT INTO munki_software (name, display_name) VALUES ($1, $1) RETURNING id`, name,
	).Scan(&softwareID); err != nil {
		t.Fatalf("insert software: %v", err)
	}
	var objectID int64
	if err := db.Pool().QueryRow(ctx,
		`INSERT INTO storage_objects (prefix, filename, content_type, size_bytes, sha256, available_at)
		 VALUES ('packages', $1, 'application/octet-stream', $2, $3, now()) RETURNING id`,
		name+".pkg", size, sha256,
	).Scan(&objectID); err != nil {
		t.Fatalf("insert object: %v", err)
	}
	var packageID int64
	if err := db.Pool().QueryRow(ctx,
		`INSERT INTO munki_packages (software_id, version, installer_object_id)
		 VALUES ($1, '1.0', $2) RETURNING id`,
		softwareID, objectID,
	).Scan(&packageID); err != nil {
		t.Fatalf("insert package: %v", err)
	}
	return packageID
}

// recordCurrent reports a package as mirrored with the given hash, the worker's
// package_current event applied server-side.
func recordCurrent(
	t *testing.T,
	store *mdp.Store,
	ctx context.Context,
	dpID, packageID int64,
	sha256 string,
) {
	t.Helper()
	if err := store.RecordPackageState(
		ctx, dpID, packageID, mdp.PackageStatusCurrent, sha256, "",
	); err != nil {
		t.Fatalf("RecordPackageState current: %v", err)
	}
}

func TestGetByIDDerivesPackageStatusFromDesiredAndMirrorState(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, _ := newStore(db)
	shaA := strings.Repeat("a", 64)
	shaB := strings.Repeat("b", 64)
	pkgA := seedAvailablePackage(t, db, ctx, "Chrome", shaA, 4096)
	pkgB := seedAvailablePackage(t, db, ctx, "Firefox", shaB, 8192)

	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "key-mel")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	states := packageStates(t, store, ctx, point.ID)
	if states[pkgA].Status != mdp.PackageStatusPending || states[pkgB].Status != mdp.PackageStatusPending {
		t.Fatalf("initial states = %+v, want both pending", states)
	}

	// pkgA reports the desired hash; pkgB reports a stale one. Each is derived
	// independently against Woodstar's current desired installer.
	recordCurrent(t, store, ctx, point.ID, pkgA, shaA)
	recordCurrent(t, store, ctx, point.ID, pkgB, strings.Repeat("c", 64))

	states = packageStates(t, store, ctx, point.ID)
	if a := states[pkgA]; a.Status != mdp.PackageStatusCurrent {
		t.Fatalf("matching hash status = %q, want current", a.Status)
	}
	if b := states[pkgB]; b.Status != mdp.PackageStatusSyncing {
		t.Fatalf("stale hash status = %q, want syncing", b.Status)
	}

	// pkgB catching up to the desired hash flips it current without touching pkgA.
	recordCurrent(t, store, ctx, point.ID, pkgB, shaB)
	states = packageStates(t, store, ctx, point.ID)
	if a := states[pkgA]; a.Status != mdp.PackageStatusCurrent {
		t.Fatalf("untouched package status = %q, want current", a.Status)
	}
	if b := states[pkgB]; b.Status != mdp.PackageStatusCurrent {
		t.Fatalf("caught-up package status = %q, want current", b.Status)
	}
}

func TestGetByIDSurfacesPackageError(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, _ := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "key-mel")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.RecordPackageState(
		ctx, point.ID, pkg, mdp.PackageStatusError, "", "package 7 verify: sha256 mismatch",
	); err != nil {
		t.Fatalf("RecordPackageState error: %v", err)
	}

	states := packageStates(t, store, ctx, point.ID)
	if got := states[pkg]; got.Status != mdp.PackageStatusError || got.Error == "" {
		t.Fatalf("error package state = %+v, want error with message", got)
	}
}

func TestListMarksOnlineFromPresence(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)

	pointID := mustCreate(t, store, ctx, "Melbourne")
	points, _, err := store.List(ctx, mdp.DistributionPointListParams{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if points[0].Online {
		t.Fatal("Online = true without presence")
	}

	presence.Connect(pointID)
	points, _, err = store.List(ctx, mdp.DistributionPointListParams{})
	if err != nil {
		t.Fatalf("List online: %v", err)
	}
	if !points[0].Online {
		t.Fatal("Online = false with presence")
	}
}

func TestResolveForClientHonorsEveryGate(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)

	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "key-mel")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recordCurrent(t, store, ctx, point.ID, pkg, sha)
	presence.Connect(point.ID)

	inside := mustAddr(t, "10.1.2.3")
	resolved, err := store.ResolveForClient(ctx, inside, pkg)
	if err != nil {
		t.Fatalf("ResolveForClient: %v", err)
	}
	if resolved == nil || resolved.ID != point.ID || resolved.Key != "key-mel" {
		t.Fatalf("resolved = %+v, want point %d with key", resolved, point.ID)
	}

	if got := resolveOrNil(t, store, ctx, mustAddr(t, "192.168.1.1"), pkg); got != nil {
		t.Fatalf("client outside CIDRs resolved %+v, want nil", got)
	}
	if got := resolveOrNil(t, store, ctx, inside, pkg+9999); got != nil {
		t.Fatalf("package not mirrored by this point resolved %+v, want nil", got)
	}

	// A stale hash means the distribution point is not current for this installer.
	if err := store.RecordPackageState(
		ctx, point.ID, pkg, mdp.PackageStatusCurrent, strings.Repeat("f", 64), "",
	); err != nil {
		t.Fatalf("stale RecordPackageState: %v", err)
	}
	if got := resolveOrNil(t, store, ctx, inside, pkg); got != nil {
		t.Fatalf("stale-hash point resolved %+v, want nil", got)
	}
	recordCurrent(t, store, ctx, point.ID, pkg, sha)

	if err := store.RecordPackageState(
		ctx, point.ID, pkg, mdp.PackageStatusError, sha, "package 7 download: unavailable",
	); err != nil {
		t.Fatalf("error RecordPackageState: %v", err)
	}
	if got := resolveOrNil(t, store, ctx, inside, pkg); got != nil {
		t.Fatalf("error package resolved %+v, want nil", got)
	}
	recordCurrent(t, store, ctx, point.ID, pkg, sha)

	presence.Disconnect(point.ID)
	if got := resolveOrNil(t, store, ctx, inside, pkg); got != nil {
		t.Fatalf("offline point resolved %+v, want nil", got)
	}
	presence.Connect(point.ID)

	if _, err := store.Update(ctx, point.ID, mdp.DistributionPointMutation{
		Name:          "Melbourne",
		Enabled:       false,
		ClientCIDRs:   []string{"10.0.0.0/8"},
		ClientBaseURL: "https://mdp.example",
	}); err != nil {
		t.Fatalf("Update disabled: %v", err)
	}
	if got := resolveOrNil(t, store, ctx, inside, pkg); got != nil {
		t.Fatalf("disabled point resolved %+v, want nil", got)
	}
}

func TestResolveForClientSkipsEmptyClientBaseURL(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)

	point, err := store.Create(ctx, mdp.DistributionPointMutation{
		Name:          "Melbourne",
		Enabled:       true,
		ClientCIDRs:   []string{"10.0.0.0/8"},
		ClientBaseURL: "",
	}, "key-mel")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recordCurrent(t, store, ctx, point.ID, pkg, sha)
	presence.Connect(point.ID)

	if got := resolveOrNil(t, store, ctx, mustAddr(t, "10.1.2.3"), pkg); got != nil {
		t.Fatalf("point with empty client_base_url resolved %+v, want nil", got)
	}
}

func TestReorderRequiresExactSet(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, _ := newStore(db)
	a := mustCreate(t, store, ctx, "A")
	b := mustCreate(t, store, ctx, "B")
	c := mustCreate(t, store, ctx, "C")

	if err := store.Reorder(ctx, []int64{a, b}); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("partial reorder error = %v, want ErrInvalidInput", err)
	}
	if err := store.Reorder(ctx, []int64{c, b, a}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}

	points, _, err := store.List(ctx, mdp.DistributionPointListParams{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	gotOrder := []int64{points[0].ID, points[1].ID, points[2].ID}
	if gotOrder[0] != c || gotOrder[1] != b || gotOrder[2] != a {
		t.Fatalf("order = %v, want %v", gotOrder, []int64{c, b, a})
	}
}

func mustCreate(t *testing.T, store *mdp.Store, ctx context.Context, name string) int64 {
	t.Helper()
	point, err := store.Create(ctx, pointMutation(name, nil), "key-"+name)
	if err != nil {
		t.Fatalf("Create %s: %v", name, err)
	}
	return point.ID
}

func mustAddr(t *testing.T, s string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("parse addr %q: %v", s, err)
	}
	return addr
}

func resolveOrNil(
	t *testing.T,
	store *mdp.Store,
	ctx context.Context,
	addr netip.Addr,
	packageID int64,
) *mdp.ResolvedPoint {
	t.Helper()
	resolved, err := store.ResolveForClient(ctx, addr, packageID)
	if err != nil {
		t.Fatalf("ResolveForClient: %v", err)
	}
	return resolved
}

func packageStates(
	t *testing.T,
	store *mdp.Store,
	ctx context.Context,
	id int64,
) map[int64]mdp.PackageState {
	t.Helper()
	detail, err := store.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	byID := make(map[int64]mdp.PackageState, len(detail.Packages))
	for _, state := range detail.Packages {
		byID[state.PackageID] = state
	}
	return byID
}
