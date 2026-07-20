//go:build postgres

package agentauth

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestAgentSecretLifecycle(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)

	orbitSecret := createAgentSecret(t, ctx, store, AgentOrbit, "orbit-secret-value-long-enough-32")
	santaSecret := createAgentSecret(t, ctx, store, AgentSanta, "santa-secret-value-long-enough-32")
	munkiSecret := createAgentSecret(t, ctx, store, AgentMunki, "munki-secret-value-long-enough-32")

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

	requireAgentSecretVerification(t, ctx, store, AgentOrbit, orbitSecret.Value, true)
	requireAgentSecretVerification(t, ctx, store, AgentSanta, orbitSecret.Value, false)
	requireAgentSecretVerification(t, ctx, store, AgentMunki, munkiSecret.Value, true)
	requireAgentSecretVerification(t, ctx, store, AgentOrbit, "", false)

	updatedOrbitSecret, err := store.Update(ctx, orbitSecret.ID, AgentSecretMutation{
		Value: "updated-orbit-secret-value-long-32",
	})
	if err != nil {
		t.Fatalf("update orbit secret: %v", err)
	}
	if updatedOrbitSecret.Value != "updated-orbit-secret-value-long-32" {
		t.Fatalf("updated orbit secret value = %q, want updated value", updatedOrbitSecret.Value)
	}
	requireAgentSecretVerification(t, ctx, store, AgentOrbit, orbitSecret.Value, false)
	requireAgentSecretVerification(t, ctx, store, AgentOrbit, updatedOrbitSecret.Value, true)
	orbitSecret = updatedOrbitSecret

	if err := store.Delete(ctx, orbitSecret.ID); err != nil {
		t.Fatalf("delete orbit secret: %v", err)
	}
	requireAgentSecretVerification(t, ctx, store, AgentOrbit, orbitSecret.Value, false)
}

func createAgentSecret(
	t *testing.T,
	ctx context.Context,
	store *Store,
	agent Agent,
	value string,
) *AgentSecret {
	t.Helper()
	secret, err := store.Create(ctx, AgentSecretCreate{Agent: agent, Value: value})
	if err != nil {
		t.Fatalf("create %s agent secret: %v", agent, err)
	}
	return secret
}

func requireAgentSecretVerification(
	t *testing.T,
	ctx context.Context,
	store *Store,
	agent Agent,
	value string,
	want bool,
) {
	t.Helper()
	got, err := store.Verify(ctx, agent, value)
	if err != nil {
		t.Fatalf("verify %s agent secret: %v", agent, err)
	}
	if got != want {
		t.Fatalf("Verify(%s) = %v, want %v", agent, got, want)
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
