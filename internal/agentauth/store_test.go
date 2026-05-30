package agentauth

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestAgentSecretLifecycle(t *testing.T) {
	database, ctx := dbtest.Open(t)
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

func TestCreateRejectsUnknownAgent(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)

	if _, err := store.Create(ctx, AgentSecretCreate{
		Agent: Agent("osquery"),
		Value: "raw-osquery-secret-value-long-32",
	}); err == nil {
		t.Fatal("Create accepted unknown agent")
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

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		want          string
		wantOK        bool
	}{
		{name: "missing", wantOK: false},
		{name: "wrong scheme", authorization: "Token abc", wantOK: false},
		{name: "empty bearer", authorization: "Bearer ", wantOK: false},
		{name: "spaces in token", authorization: "Bearer abc def", wantOK: false},
		{name: "valid", authorization: "Bearer abc", want: "abc", wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BearerToken(tt.authorization)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("BearerToken() = %q, %v; want %q, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
