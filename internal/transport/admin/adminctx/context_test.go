package adminctx

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/db/sqlc"
	"github.com/woodleighschool/woodstar/internal/models"
)

func TestUserFromContextRoundTrip(t *testing.T) {
	user := &models.User{User: sqlc.User{ID: 7, Role: models.RoleAdmin}}
	ctx := WithUser(context.Background(), user)

	got, ok := UserFromContext(ctx)
	if !ok {
		t.Fatal("UserFromContext returned ok=false")
	}
	if got != user {
		t.Fatalf("got %v, want %v", got, user)
	}
}

func TestUserFromContextEmpty(t *testing.T) {
	if _, ok := UserFromContext(context.Background()); ok {
		t.Fatal("expected ok=false on bare context")
	}
}

func TestUserFromContextRejectsNilUser(t *testing.T) {
	ctx := WithUser(context.Background(), nil)
	if _, ok := UserFromContext(ctx); ok {
		t.Fatal("expected ok=false when nil user is stored")
	}
}
