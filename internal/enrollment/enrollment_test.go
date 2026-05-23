package enrollment

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateNodeKeyLengthAndAlphabet(t *testing.T) {
	for range 32 {
		key, err := GenerateNodeKey()
		if err != nil {
			t.Fatalf("GenerateNodeKey returned error: %v", err)
		}
		if len(key) != NodeKeyLength {
			t.Fatalf("len = %d, want %d", len(key), NodeKeyLength)
		}
		for _, r := range key {
			if !strings.ContainsRune(NodeKeyAlphabet, r) {
				t.Fatalf("rune %q outside expected alphabet", r)
			}
		}
	}
}

func TestGenerateNodeKeyIsRandom(t *testing.T) {
	seen := map[string]bool{}
	for range 64 {
		key, err := GenerateNodeKey()
		if err != nil {
			t.Fatalf("GenerateNodeKey returned error: %v", err)
		}
		if seen[key] {
			t.Fatalf("duplicate key %q produced within 64 attempts", key)
		}
		seen[key] = true
	}
}

func TestEnrollmentErrorsAreSharedSentinels(t *testing.T) {
	if !errors.Is(ErrInvalidEnrollSecret, ErrInvalidEnrollSecret) {
		t.Fatal("ErrInvalidEnrollSecret is not a sentinel")
	}
	if !errors.Is(ErrMissingHardwareUUID, ErrMissingHardwareUUID) {
		t.Fatal("ErrMissingHardwareUUID is not a sentinel")
	}
}
