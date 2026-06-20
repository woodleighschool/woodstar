package mdp_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/munki/mdp/grant"
)

func TestSelectRedirectMintsVerifiableGrant(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "sel-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recordCurrent(t, store, ctx, point.ID, pkg, sha)
	presence.Connect(point.ID)

	redirect, ok := store.SelectRedirect(ctx, mdp.SelectionRequest{
		ClientIP:              "10.1.2.3",
		PackageID:             pkg,
		InstallerItemLocation: "packages/20/installer/Chrome.pkg",
		SHA256:                sha,
		SizeBytes:             4096,
	})
	if !ok {
		t.Fatal("SelectRedirect returned no match, want redirect")
	}

	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if parsed.Path != "/munki/pkgs/packages/20/installer/Chrome.pkg" {
		t.Fatalf("redirect path = %q, want package path", parsed.Path)
	}
	claims, err := grant.Verify([]byte("sel-key"), parsed.Query().Get("cap"), time.Now())
	if err != nil {
		t.Fatalf("grant.Verify: %v", err)
	}
	if claims.PackageID != pkg || claims.SizeBytes != 4096 || claims.SHA256 != sha {
		t.Fatalf("grant integrity claims = %+v, want package %d", claims, pkg)
	}
	if claims.InstallerItemLocation != "packages/20/installer/Chrome.pkg" {
		t.Fatalf("grant installer_item_location = %q", claims.InstallerItemLocation)
	}
	if claims.DistributionPointID != point.ID {
		t.Fatalf("grant distribution_point_id = %d, want %d", claims.DistributionPointID, point.ID)
	}
}

func TestSelectRedirectFallsBackWithoutEligiblePoint(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store, presence := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, db, ctx, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "sel-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recordCurrent(t, store, ctx, point.ID, pkg, sha)
	presence.Connect(point.ID)
	if _, ok := store.SelectRedirect(
		ctx,
		mdp.SelectionRequest{PackageID: pkg, SHA256: sha, SizeBytes: 4096},
	); ok {
		t.Fatal("SelectRedirect matched without a client IP, want fallback")
	}
	if _, ok := store.SelectRedirect(ctx, mdp.SelectionRequest{
		ClientIP: "192.168.1.1", PackageID: pkg, SHA256: sha, SizeBytes: 4096,
	}); ok {
		t.Fatal("SelectRedirect matched a client outside all CIDRs, want fallback")
	}
}
