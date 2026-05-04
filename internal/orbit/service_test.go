package orbit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateNodeKeyLengthAndAlphabet(t *testing.T) {
	for range 32 {
		key, err := generateNodeKey()
		if err != nil {
			t.Fatalf("generateNodeKey returned error: %v", err)
		}
		if len(key) != nodeKeyLength {
			t.Fatalf("len = %d, want %d", len(key), nodeKeyLength)
		}
		for _, r := range key {
			if !strings.ContainsRune(nodeKeyAlphabet, r) {
				t.Fatalf("rune %q outside expected alphabet", r)
			}
		}
	}
}

func TestGenerateNodeKeyIsRandom(t *testing.T) {
	seen := map[string]bool{}
	for range 64 {
		key, err := generateNodeKey()
		if err != nil {
			t.Fatalf("generateNodeKey returned error: %v", err)
		}
		if seen[key] {
			t.Fatalf("duplicate key %q produced within 64 attempts", key)
		}
		seen[key] = true
	}
}

func TestConfigResponseWireShapeMatchesOrbit(t *testing.T) {
	body, err := json.Marshal(ConfigResponse{
		Flags:         []byte("{}"),
		Notifications: map[string]any{},
	})
	if err != nil {
		t.Fatalf("marshal config response: %v", err)
	}

	got := string(body)
	if !strings.Contains(got, `"command_line_startup_flags":{}`) {
		t.Fatalf("config response = %s, missing command_line_startup_flags object", got)
	}
	if strings.Contains(got, `"extensions":[]`) {
		t.Fatalf("config response = %s, extensions must not be an array", got)
	}
}
