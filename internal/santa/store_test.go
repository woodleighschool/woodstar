package santa_test

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/santa"
)

func TestSyncTokenLifecycleAndBearerVerification(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := santa.NewStore(db)

	created, err := store.CreateSyncToken(ctx)
	if err != nil {
		t.Fatalf("create sync token: %v", err)
	}
	if created.Value == "" {
		t.Fatal("created token value is empty")
	}
	if created.ValueHash == "" {
		t.Fatal("created token hash is empty")
	}
	if created.ValueHash == created.Value {
		t.Fatal("created token stored plaintext as hash")
	}

	tokens, err := store.ListSyncTokens(ctx)
	if err != nil {
		t.Fatalf("list sync tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("token count = %d, want 1", len(tokens))
	}
	if tokens[0].ValueHash != created.ValueHash {
		t.Fatalf("listed hash = %q, want %q", tokens[0].ValueHash, created.ValueHash)
	}
	if tokens[0].LastUsedAt != nil {
		t.Fatalf("last used = %v, want nil before verification", tokens[0].LastUsedAt)
	}

	verifyCases := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "missing", header: "", want: false},
		{name: "wrong scheme", header: "Token " + created.Value, want: false},
		{name: "unknown token", header: "Bearer not-the-token", want: false},
		{name: "known token", header: "Bearer " + created.Value, want: true},
	}
	for _, tt := range verifyCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.VerifyBearerToken(context.Background(), tt.header)
			if err != nil {
				t.Fatalf("verify bearer token: %v", err)
			}
			if got != tt.want {
				t.Fatalf("verified = %v, want %v", got, tt.want)
			}
		})
	}

	tokens, err = store.ListSyncTokens(ctx)
	if err != nil {
		t.Fatalf("list sync tokens after verification: %v", err)
	}
	if tokens[0].LastUsedAt == nil {
		t.Fatal("last used was not updated after a valid token")
	}

	if err := store.DeleteSyncToken(ctx, created.ID); err != nil {
		t.Fatalf("delete sync token: %v", err)
	}
	verified, err := store.VerifyBearerToken(ctx, "Bearer "+created.Value)
	if err != nil {
		t.Fatalf("verify deleted token: %v", err)
	}
	if verified {
		t.Fatal("deleted token still verifies")
	}
}
