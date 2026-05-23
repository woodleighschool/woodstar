package enrollment

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestEnrollSecretLifecycle(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)

	secret, err := store.Create(ctx)
	if err != nil {
		t.Fatalf("create enroll secret: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Delete(context.Background(), secret.ID); err != nil {
			t.Logf("cleanup enroll secret: %v", err)
		}
	})

	secrets, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list enroll secrets: %v", err)
	}
	if !containsSecretValue(secrets, secret.Value) {
		t.Fatalf("created secret %q not found in list", secret.Value)
	}

	ok, err := store.HasActive(ctx, secret.Value)
	if err != nil {
		t.Fatalf("check enroll secret: %v", err)
	}
	if !ok {
		t.Fatal("created secret not active")
	}

	if err := store.Delete(ctx, secret.ID); err != nil {
		t.Fatalf("delete enroll secret: %v", err)
	}

	ok, err = store.HasActive(ctx, secret.Value)
	if err != nil {
		t.Fatalf("check deleted enroll secret: %v", err)
	}
	if ok {
		t.Fatal("deleted secret is still active")
	}
}

func containsSecretValue(secrets []EnrollSecret, value string) bool {
	for _, secret := range secrets {
		if secret.Value == value {
			return true
		}
	}
	return false
}
