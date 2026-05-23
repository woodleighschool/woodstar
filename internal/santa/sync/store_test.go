package sync_test

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	santasync "github.com/woodleighschool/woodstar/internal/santa/sync"
)

func TestSyncTokenLifecycleAndVerification(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := santasync.NewStore(db)

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
