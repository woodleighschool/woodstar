package orbit

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

// ErrMissingHardwareUUID reports an enrollment request without a host identity.
var ErrMissingHardwareUUID = errors.New("hardware_uuid is required")

// SecretVerifier checks whether an enrollment secret is active for an agent.
type SecretVerifier interface {
	Verify(context.Context, agentauth.Agent, string) (bool, error)
}

// IssueNodeKey verifies the Orbit enrollment secret and returns a fresh node key.
func IssueNodeKey(ctx context.Context, verifier SecretVerifier, enrollSecret string) (string, error) {
	ok, err := verifier.Verify(ctx, agentauth.AgentOrbit, enrollSecret)
	if err != nil {
		return "", fmt.Errorf("validate enroll secret: %w", err)
	}
	if !ok {
		return "", agentauth.ErrInvalidSecret
	}

	nodeKey, err := randtoken.Generate(32)
	if err != nil {
		return "", fmt.Errorf("generate node key: %w", err)
	}
	return nodeKey, nil
}
