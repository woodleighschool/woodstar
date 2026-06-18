package mdp_test

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
)

// presenceStub is a settable presence set for resolver tests.
type presenceStub map[int64]bool

func (p presenceStub) Online(id int64) bool { return p[id] }

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
		`INSERT INTO munki_software (name) VALUES ($1) RETURNING id`, name,
	).Scan(&softwareID); err != nil {
		t.Fatalf("insert software: %v", err)
	}
	var objectID int64
	if err := db.Pool().QueryRow(ctx,
		`INSERT INTO storage_objects (prefix, filename, size_bytes, sha256, available_at)
		 VALUES ('packages', $1, $2, $3, now()) RETURNING id`,
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

func verifiedReport(packageID int64, sha256 string) mdp.StateReport {
	return mdp.StateReport{
		Packages: []mdp.ReportedPackage{
			{PackageID: packageID, SHA256: sha256, Status: mdp.PackageStatusCurrent},
		},
	}
}

func TestGetByIDDerivesPackageStatusFromDesiredAndMirrorState(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := mdp.NewStore(db)
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

	if err := store.RecordState(ctx, point.ID, mdp.StateReport{
		Packages: []mdp.ReportedPackage{
			{PackageID: pkgA, SHA256: shaA, Status: mdp.PackageStatusCurrent},
			{PackageID: pkgB, SHA256: strings.Repeat("c", 64), Status: mdp.PackageStatusCurrent},
		},
	}); err != nil {
		t.Fatalf("RecordState: %v", err)
	}

	states = packageStates(t, store, ctx, point.ID)
	if a := states[pkgA]; a.Status != mdp.PackageStatusCurrent {
		t.Fatalf("matching hash status = %q, want current", a.Status)
	}
	if b := states[pkgB]; b.Status != mdp.PackageStatusSyncing {
		t.Fatalf("stale hash status = %q, want syncing", b.Status)
	}

	if err := store.RecordState(ctx, point.ID, verifiedReport(pkgA, shaA)); err != nil {
		t.Fatalf("second RecordState: %v", err)
	}
	states = packageStates(t, store, ctx, point.ID)
	if b := states[pkgB]; b.Status != mdp.PackageStatusPending {
		t.Fatalf("missing report status = %q, want pending", b.Status)
	}
	if a := states[pkgA]; a.Status != mdp.PackageStatusCurrent {
		t.Fatalf("reported package status = %q, want current", a.Status)
	}
}

func TestGetByIDSurfacesPackageError(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := mdp.NewStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "key-mel")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.RecordState(ctx, point.ID, mdp.StateReport{
		Packages: []mdp.ReportedPackage{
			{
				PackageID: pkg,
				Status:    mdp.PackageStatusError,
				Error:     "package 7 verify: sha256 mismatch",
			},
		},
	}); err != nil {
		t.Fatalf("RecordState: %v", err)
	}

	states := packageStates(t, store, ctx, point.ID)
	if got := states[pkg]; got.Status != mdp.PackageStatusError || got.Error == "" {
		t.Fatalf("error package state = %+v, want error with message", got)
	}
}

func TestListMarksOnlineFromPresence(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := mdp.NewStore(db)
	online := presenceStub{}
	store.SetPresence(online)

	pointID := mustCreate(t, store, ctx, "Melbourne")
	points, _, err := store.List(ctx, mdp.DistributionPointListParams{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if points[0].Online {
		t.Fatal("Online = true without presence")
	}

	online[pointID] = true
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
	store := mdp.NewStore(db)
	online := presenceStub{}
	store.SetPresence(online)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)

	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "key-mel")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.RecordState(ctx, point.ID, verifiedReport(pkg, sha)); err != nil {
		t.Fatalf("RecordState: %v", err)
	}
	online[point.ID] = true

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

	// A stale hash (installer changed, DP not current yet) is not eligible.
	if err := store.RecordState(ctx, point.ID, mdp.StateReport{
		Packages: []mdp.ReportedPackage{
			{PackageID: pkg, SHA256: strings.Repeat("f", 64), Status: mdp.PackageStatusCurrent},
		},
	}); err != nil {
		t.Fatalf("stale RecordState: %v", err)
	}
	if got := resolveOrNil(t, store, ctx, inside, pkg); got != nil {
		t.Fatalf("stale-hash point resolved %+v, want nil", got)
	}
	if err := store.RecordState(ctx, point.ID, verifiedReport(pkg, sha)); err != nil {
		t.Fatalf("restore current package: %v", err)
	}

	if err := store.RecordState(ctx, point.ID, mdp.StateReport{
		Packages: []mdp.ReportedPackage{
			{
				PackageID: pkg,
				SHA256:    sha,
				Status:    mdp.PackageStatusError,
				Error:     "package 7 download: unavailable",
			},
		},
	}); err != nil {
		t.Fatalf("error RecordState: %v", err)
	}
	if got := resolveOrNil(t, store, ctx, inside, pkg); got != nil {
		t.Fatalf("error package resolved %+v, want nil", got)
	}
	if err := store.RecordState(ctx, point.ID, verifiedReport(pkg, sha)); err != nil {
		t.Fatalf("restore current package after error: %v", err)
	}

	delete(online, point.ID)
	if got := resolveOrNil(t, store, ctx, inside, pkg); got != nil {
		t.Fatalf("offline point resolved %+v, want nil", got)
	}
	online[point.ID] = true

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
	store := mdp.NewStore(db)
	online := presenceStub{}
	store.SetPresence(online)
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
	if err := store.RecordState(ctx, point.ID, verifiedReport(pkg, sha)); err != nil {
		t.Fatalf("RecordState: %v", err)
	}
	online[point.ID] = true

	if got := resolveOrNil(t, store, ctx, mustAddr(t, "10.1.2.3"), pkg); got != nil {
		t.Fatalf("point with empty client_base_url resolved %+v, want nil", got)
	}
}

func TestReorderRequiresExactSet(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := mdp.NewStore(db)
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
