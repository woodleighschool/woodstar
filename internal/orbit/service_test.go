package orbit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfigResponseWireShapeMatchesOrbit(t *testing.T) {
	body, err := json.Marshal(ConfigResponse{})
	if err != nil {
		t.Fatalf("marshal config response: %v", err)
	}

	got := string(body)
	if got != "{}" {
		t.Fatalf("config response = %s, want empty object", got)
	}
	if strings.Contains(got, "command_line_startup_flags") {
		t.Fatalf("config response = %s, command_line_startup_flags is not configured yet", got)
	}
}
