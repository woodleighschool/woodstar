//go:build e2e

package e2e

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

func loadProtocolFixture(
	t *testing.T,
	fixtures fs.FS,
	protocol string,
	name string,
	replacements map[string]any,
) []byte {
	t.Helper()

	path := "testdata/" + protocol + "/" + name
	payload, err := fs.ReadFile(fixtures, path)
	if err != nil {
		t.Fatalf("read %s protocol fixture %s: %v", protocol, name, err)
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatalf("decode %s protocol fixture %s: %v", protocol, name, err)
	}
	value = replaceProtocolFixtureValues(t, protocol, name, value, replacements)
	payload, err = json.Marshal(value)
	if err != nil {
		t.Fatalf("encode %s protocol fixture %s: %v", protocol, name, err)
	}
	return payload
}

func replaceProtocolFixtureValues(
	t *testing.T,
	protocol string,
	name string,
	value any,
	replacements map[string]any,
) any {
	t.Helper()

	switch value := value.(type) {
	case map[string]any:
		replaced := make(map[string]any, len(value))
		for key, child := range value {
			if strings.HasPrefix(key, "$") {
				replacement, ok := replacements[key]
				if !ok {
					t.Fatalf("%s protocol fixture %s has no replacement for key %s", protocol, name, key)
				}
				replacementKey, ok := replacement.(string)
				if !ok || replacementKey == "" {
					t.Fatalf(
						"%s protocol fixture %s key %q replacement is not a non-empty string",
						protocol,
						name,
						replacement,
					)
				}
				key = replacementKey
			}
			replaced[key] = replaceProtocolFixtureValues(t, protocol, name, child, replacements)
		}
		return replaced
	case []any:
		for i := range value {
			value[i] = replaceProtocolFixtureValues(t, protocol, name, value[i], replacements)
		}
		return value
	case string:
		if !strings.HasPrefix(value, "$") {
			return value
		}
		replacement, ok := replacements[value]
		if !ok {
			t.Fatalf("%s protocol fixture %s has no replacement for value %s", protocol, name, value)
		}
		return replacement
	default:
		return value
	}
}
