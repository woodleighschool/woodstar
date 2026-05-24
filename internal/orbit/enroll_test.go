package orbit

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/agentauth"
)

func TestIssueNodeKeyUsesOrbitAgentSecret(t *testing.T) {
	verifier := &recordingSecretVerifier{valid: true}

	nodeKey, err := IssueNodeKey(context.Background(), verifier, "orbit-secret")
	if err != nil {
		t.Fatalf("IssueNodeKey returned error: %v", err)
	}
	if nodeKey == "" {
		t.Fatal("IssueNodeKey returned empty node key")
	}
	if verifier.agent != agentauth.AgentOrbit {
		t.Fatalf("verified agent = %q, want %q", verifier.agent, agentauth.AgentOrbit)
	}
	if verifier.value != "orbit-secret" {
		t.Fatalf("verified value = %q, want orbit-secret", verifier.value)
	}
}

func TestIssueNodeKeyRejectsInvalidSecret(t *testing.T) {
	_, err := IssueNodeKey(context.Background(), &recordingSecretVerifier{}, "wrong")
	if !errors.Is(err, agentauth.ErrInvalidSecret) {
		t.Fatalf("error = %v, want ErrInvalidSecret", err)
	}
}

func TestIssueNodeKeyPropagatesVerifierError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	_, err := IssueNodeKey(context.Background(), &recordingSecretVerifier{err: wantErr}, "orbit-secret")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

type recordingSecretVerifier struct {
	agent agentauth.Agent
	value string
	valid bool
	err   error
}

func (v *recordingSecretVerifier) Verify(_ context.Context, agent agentauth.Agent, value string) (bool, error) {
	v.agent = agent
	v.value = value
	return v.valid, v.err
}
