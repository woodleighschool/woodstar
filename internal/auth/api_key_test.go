package auth

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/randtoken"
)

func TestGenerateAPIKey(t *testing.T) {
	t.Parallel()
	a, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		t.Fatalf("randtoken.Generate returned error: %v", err)
	}
	b, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		t.Fatalf("randtoken.Generate returned error: %v", err)
	}
	if a == b {
		t.Fatalf("two consecutive keys collided: %q", a)
	}
	if len(a) < 32 {
		t.Fatalf("key length = %d, want >= 32 (24 random bytes base64-encoded)", len(a))
	}
}
