package syncstate_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func TestSyncTokenLifecycleAndVerification(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)

	created, err := store.CreateToken(ctx)
	if err != nil {
		t.Fatalf("create sync token: %v", err)
	}
	if created.Value == "" {
		t.Fatal("created token value is empty")
	}

	tokens, err := store.ListTokens(ctx)
	if err != nil {
		t.Fatalf("list sync tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("token count = %d, want 1", len(tokens))
	}
	if tokens[0].Value != created.Value {
		t.Fatalf("listed value = %q, want created value", tokens[0].Value)
	}

	verifyCases := []struct {
		name  string
		token string
		want  bool
	}{
		{name: "missing", token: "", want: false},
		{name: "unknown token", token: "not-the-token", want: false},
		{name: "known token", token: created.Value, want: true},
	}
	for _, tt := range verifyCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.VerifySyncToken(context.Background(), tt.token)
			if err != nil {
				t.Fatalf("verify sync token: %v", err)
			}
			if got != tt.want {
				t.Fatalf("verified = %v, want %v", got, tt.want)
			}
		})
	}

	if err := store.DeleteToken(ctx, created.ID); err != nil {
		t.Fatalf("delete sync token: %v", err)
	}
	verified, err := store.VerifySyncToken(ctx, created.Value)
	if err != nil {
		t.Fatalf("verify deleted token: %v", err)
	}
	if verified {
		t.Fatal("deleted token still verifies")
	}
}

func TestLoadPendingTargetsPagePaginatesDeterministically(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)
	host := createHost(t, ctx, db)

	targets := []syncstate.Target{
		{RuleType: "binary", Identifier: "a", Policy: "blocklist", PayloadHash: "a"},
		{RuleType: "binary", Identifier: "b", Policy: "blocklist", PayloadHash: "b"},
		{RuleType: "binary", Identifier: "c", Policy: "blocklist", PayloadHash: "c"},
	}
	if err := store.ReplacePending(ctx, host.ID, "client-hash", targets, targets, true); err != nil {
		t.Fatalf("replace pending: %v", err)
	}

	first, err := store.LoadPendingTargetsPage(ctx, host.ID, "", 2)
	if err != nil {
		t.Fatalf("load first page: %v", err)
	}
	if first.Cursor == "" {
		t.Fatal("first page cursor is empty")
	}
	if got := targetIdentifiers(first.Targets); got != "a,b" {
		t.Fatalf("first page identifiers = %q, want a,b", got)
	}

	second, err := store.LoadPendingTargetsPage(ctx, host.ID, first.Cursor, 2)
	if err != nil {
		t.Fatalf("load second page: %v", err)
	}
	if second.Cursor != "" {
		t.Fatalf("second page cursor = %q, want empty", second.Cursor)
	}
	if got := targetIdentifiers(second.Targets); got != "c" {
		t.Fatalf("second page identifiers = %q, want c", got)
	}
}

func TestLoadPendingTargetsPageRejectsInvalidCursor(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := syncstate.NewStore(db)

	_, err := store.LoadPendingTargetsPage(ctx, 1, "not-base64", 2)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func targetIdentifiers(targets []syncstate.Target) string {
	out := ""
	var outSb120 strings.Builder
	for i, target := range targets {
		if i > 0 {
			outSb120.WriteString(",")
		}
		outSb120.WriteString(target.Identifier)
	}
	out += outSb120.String()
	return out
}

func createHost(t *testing.T, ctx context.Context, db *database.DB) *hosts.Host {
	t.Helper()

	host, err := hosts.NewStore(db).UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "syncstate-page-host",
		OrbitNodeKey: "syncstate-page-orbit",
	})
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	return host
}
