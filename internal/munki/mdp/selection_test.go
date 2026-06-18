package mdp_test

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/munki/mdp/grant"
)

func TestSelectRedirectMintsVerifiableGrant(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := mdp.NewStore(db)
	online := presenceStub{}
	store.SetPresence(online)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "sel-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.RecordState(ctx, point.ID, verifiedReport(pkg, sha)); err != nil {
		t.Fatalf("RecordState: %v", err)
	}
	online[point.ID] = true

	selection := mdp.NewSelection(store, discardLogger())
	redirect, ok := selection.SelectRedirect(ctx, mdp.SelectionRequest{
		ClientIP:  "10.1.2.3",
		HostID:    7,
		Serial:    "C02ABC",
		PackageID: pkg,
		SHA256:    sha,
		SizeBytes: 4096,
	})
	if !ok {
		t.Fatal("SelectRedirect returned no match, want redirect")
	}

	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if !strings.HasSuffix(parsed.Path, "/munki-distribution/packages/"+strconv.FormatInt(pkg, 10)) {
		t.Fatalf("redirect path = %q, want package path", parsed.Path)
	}
	claims, err := grant.Verify([]byte("sel-key"), parsed.Query().Get("cap"), time.Now())
	if err != nil {
		t.Fatalf("grant.Verify: %v", err)
	}
	if claims.PackageID != pkg || claims.SizeBytes != 4096 || claims.SHA256 != sha {
		t.Fatalf("grant integrity claims = %+v, want package %d", claims, pkg)
	}
	if claims.DistributionPointID != point.ID || claims.HostID != 7 || claims.Serial != "C02ABC" {
		t.Fatalf("grant audit claims = %+v", claims)
	}
}

func TestSelectRedirectFallsBackWithoutEligiblePoint(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := mdp.NewStore(db)
	online := presenceStub{}
	store.SetPresence(online)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "sel-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.RecordState(ctx, point.ID, verifiedReport(pkg, sha)); err != nil {
		t.Fatalf("RecordState: %v", err)
	}
	online[point.ID] = true
	selection := mdp.NewSelection(store, discardLogger())

	if _, ok := selection.SelectRedirect(
		ctx,
		mdp.SelectionRequest{PackageID: pkg, SHA256: sha, SizeBytes: 4096},
	); ok {
		t.Fatal("SelectRedirect matched without a client IP, want fallback")
	}
	if _, ok := selection.SelectRedirect(ctx, mdp.SelectionRequest{
		ClientIP: "192.168.1.1", PackageID: pkg, SHA256: sha, SizeBytes: 4096,
	}); ok {
		t.Fatal("SelectRedirect matched a client outside all CIDRs, want fallback")
	}
}
