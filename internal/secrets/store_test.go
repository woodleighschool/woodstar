package secrets

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/db/dbtest"
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

	validated, ok, err := store.ValidateOrbitEnrollSecret(ctx, secret.Value)
	if err != nil {
		t.Fatalf("validate orbit enroll secret: %v", err)
	}
	if !ok || validated.ID != secret.ID {
		t.Fatalf("validated secret = %#v, want id %d", validated, secret.ID)
	}

	if err := store.DeleteOrbitEnrollSecret(ctx, secret.ID); err != nil {
		t.Fatalf("delete orbit enroll secret: %v", err)
	}

	validated, ok, err = store.ValidateOrbitEnrollSecret(ctx, secret.Value)
	if err != nil {
		t.Fatalf("validate deleted orbit enroll secret: %v", err)
	}
	if ok {
		t.Fatalf("validated deleted secret = %#v, want not found", validated)
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
