//go:build postgres

package mdp_test

import (
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestSelectRedirectFallsBackWithoutEligiblePoint(t *testing.T) {
	db, ctx := testdb.Open(t)
	store, presence := newStore(db)
	sha := strings.Repeat("a", 64)
	pkg := seedAvailablePackage(t, ctx, db, "Chrome", sha, 4096)
	point, err := store.Create(ctx, pointMutation("Melbourne", []string{"10.0.0.0/8"}), "sel-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recordCurrent(t, ctx, store, point.ID, pkg, sha)
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
