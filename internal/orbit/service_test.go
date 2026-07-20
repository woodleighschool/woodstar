package orbit

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestConfigResponseWireShapeMatchesOrbit(t *testing.T) {
	body, err := json.Marshal(ConfigResponse{
		CommandLineStartupFlags: json.RawMessage(orbitCommandLineStartupFlags),
	})
	if err != nil {
		t.Fatalf("marshal config response: %v", err)
	}

	var got struct {
		CommandLineStartupFlags map[string]any `json:"command_line_startup_flags"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal config response: %v", err)
	}
	flags := got.CommandLineStartupFlags
	if flags["disable_carver"] != true ||
		flags["carver_disable_function"] != true ||
		flags["logger_min_status"] != float64(4) {
		t.Fatalf("command-line flags = %#v", flags)
	}
}

func TestValidateDeviceAuthToken(t *testing.T) {
	t.Parallel()

	const valid = "11111111-2222-4333-8444-555555555555"
	if err := validateDeviceAuthToken(valid); err != nil {
		t.Fatalf("validate canonical token: %v", err)
	}

	for name, token := range map[string]string{
		"blank":           "",
		"not hexadecimal": "zzzzzzzz-2222-4333-8444-555555555555",
		"missing hyphens": "11111111222243338444555555555555",
		"uppercase":       "AAAAAAAA-BBBB-4CCC-8DDD-EEEEEEEEEEEE",
		"too long":        valid + "0",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := validateDeviceAuthToken(token); !errors.Is(err, ErrInvalidDeviceAuthToken) {
				t.Fatalf("validate token error = %v, want ErrInvalidDeviceAuthToken", err)
			}
		})
	}
}
