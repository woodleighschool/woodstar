package auth

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/secret"
)

func TestGenerateAPIKey(t *testing.T) {
	t.Parallel()
	a, err := secret.Generate(apiKeyByteLen)
	if err != nil {
		t.Fatalf("secret.Generate returned error: %v", err)
	}
	b, err := secret.Generate(apiKeyByteLen)
	if err != nil {
		t.Fatalf("secret.Generate returned error: %v", err)
	}
	if a == b {
		t.Fatalf("two consecutive keys collided: %q", a)
	}
	if len(a) < 32 {
		t.Fatalf("key length = %d, want >= 32 (24 random bytes base64-encoded)", len(a))
	}
}
