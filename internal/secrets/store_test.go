package secrets

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestOrbitEnrollSecretLifecycle(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)

	secret, err := store.CreateOrbitEnrollSecret(ctx)
	if err != nil {
		t.Fatalf("create orbit enroll secret: %v", err)
	}
	t.Cleanup(func() {
		if err := store.DeleteOrbitEnrollSecret(context.Background(), secret.ID); err != nil {
			t.Logf("cleanup orbit enroll secret: %v", err)
		}
	})

	secrets, err := store.ListOrbitEnrollSecrets(ctx)
	if err != nil {
		t.Fatalf("list orbit enroll secrets: %v", err)
	}
	if !containsSecretValue(secrets, secret.Value) {
		t.Fatalf("created secret %q not found in list", secret.Value)
	}

	ok, err := store.HasActiveOrbitEnrollSecret(ctx, secret.Value)
	if err != nil {
		t.Fatalf("check orbit enroll secret: %v", err)
	}
	if !ok {
		t.Fatal("created secret not active")
	}

	if err := store.DeleteOrbitEnrollSecret(ctx, secret.ID); err != nil {
		t.Fatalf("delete orbit enroll secret: %v", err)
	}

	ok, err = store.HasActiveOrbitEnrollSecret(ctx, secret.Value)
	if err != nil {
		t.Fatalf("check deleted orbit enroll secret: %v", err)
	}
	if ok {
		t.Fatal("deleted secret is still active")
	}
}

func containsSecretValue(secrets []Secret, value string) bool {
	for _, secret := range secrets {
		if secret.Value == value {
			return true
		}
	}
	return false
}
