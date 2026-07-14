package orbit

import (
	"encoding/json"
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
