// Package enrollment owns shared host-enrollment primitives.
package enrollment

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

// ErrMissingHardwareUUID reports an enrollment request without a host identity.
var ErrMissingHardwareUUID = errors.New("hardware_uuid is required")

const nodeKeyByteLen = 32

// IssueNodeKey verifies the shared Orbit enrollment secret and returns a fresh node key.
func IssueNodeKey(ctx context.Context, verifier agentauth.SecretVerifier, enrollSecret string) (string, error) {
	ok, err := verifier.Verify(ctx, agentauth.AgentOrbit, enrollSecret)
	if err != nil {
		return "", fmt.Errorf("validate enroll secret: %w", err)
	}
	if !ok {
		return "", agentauth.ErrInvalidSecret
	}

	nodeKey, err := randtoken.Generate(nodeKeyByteLen)
	if err != nil {
		return "", fmt.Errorf("generate node key: %w", err)
	}
	return nodeKey, nil
}
