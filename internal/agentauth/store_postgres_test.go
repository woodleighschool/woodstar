//go:build postgres

package agentauth

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestAgentSecretLifecycle(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)

	orbitSecret, err := store.Create(ctx, AgentSecretCreate{
		Agent: AgentOrbit,
		Value: "orbit-secret-value-long-enough-32",
	})
	if err != nil {
		t.Fatalf("create orbit agent secret: %v", err)
	}
	santaSecret, err := store.Create(ctx, AgentSecretCreate{
		Agent: AgentSanta,
		Value: "santa-secret-value-long-enough-32",
	})
	if err != nil {
		t.Fatalf("create santa agent secret: %v", err)
	}
	munkiSecret, err := store.Create(ctx, AgentSecretCreate{
		Agent: AgentMunki,
		Value: "munki-secret-value-long-enough-32",
	})
	if err != nil {
		t.Fatalf("create munki agent secret: %v", err)
	}

	secrets, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list agent secrets: %v", err)
	}
	if !containsAgentSecret(secrets, orbitSecret.ID, AgentOrbit, orbitSecret.Value) {
		t.Fatalf("orbit secret not listed: %+v", secrets)
	}
	if !containsAgentSecret(secrets, santaSecret.ID, AgentSanta, santaSecret.Value) {
		t.Fatalf("santa secret not listed: %+v", secrets)
	}
	if !containsAgentSecret(secrets, munkiSecret.ID, AgentMunki, munkiSecret.Value) {
		t.Fatalf("munki secret not listed: %+v", secrets)
	}

	ok, err := store.Verify(ctx, AgentOrbit, orbitSecret.Value)
	if err != nil {
		t.Fatalf("verify orbit secret: %v", err)
	}
	if !ok {
		t.Fatal("created orbit secret did not verify")
	}

	ok, err = store.Verify(ctx, AgentSanta, orbitSecret.Value)
	if err != nil {
		t.Fatalf("verify orbit secret as santa: %v", err)
	}
	if ok {
		t.Fatal("orbit secret verified for santa")
	}

	ok, err = store.Verify(ctx, AgentMunki, munkiSecret.Value)
	if err != nil {
		t.Fatalf("verify munki secret: %v", err)
	}
	if !ok {
		t.Fatal("created munki secret did not verify")
	}

	ok, err = store.Verify(ctx, AgentOrbit, "")
	if err != nil {
		t.Fatalf("verify empty secret: %v", err)
	}
	if ok {
		t.Fatal("empty secret verified")
	}

	updatedOrbitSecret, err := store.Update(ctx, orbitSecret.ID, AgentSecretMutation{
		Value: "updated-orbit-secret-value-long-32",
	})
	if err != nil {
		t.Fatalf("update orbit secret: %v", err)
	}
	if updatedOrbitSecret.Value != "updated-orbit-secret-value-long-32" {
		t.Fatalf("updated orbit secret value = %q, want updated value", updatedOrbitSecret.Value)
	}
	ok, err = store.Verify(ctx, AgentOrbit, orbitSecret.Value)
	if err != nil {
		t.Fatalf("verify old orbit secret: %v", err)
	}
	if ok {
		t.Fatal("old orbit secret still verifies after update")
	}
	ok, err = store.Verify(ctx, AgentOrbit, updatedOrbitSecret.Value)
	if err != nil {
		t.Fatalf("verify updated orbit secret: %v", err)
	}
	if !ok {
		t.Fatal("updated orbit secret did not verify")
	}
	orbitSecret = updatedOrbitSecret

	if err := store.Delete(ctx, orbitSecret.ID); err != nil {
		t.Fatalf("delete orbit secret: %v", err)
	}
	ok, err = store.Verify(ctx, AgentOrbit, orbitSecret.Value)
	if err != nil {
		t.Fatalf("verify deleted orbit secret: %v", err)
	}
	if ok {
		t.Fatal("deleted orbit secret still verifies")
	}
}

func containsAgentSecret(secrets []AgentSecret, id int64, agent Agent, value string) bool {
	for _, secret := range secrets {
		if secret.ID == id && secret.Agent == agent && secret.Value == value {
			return true
		}
	}
	return false
}
