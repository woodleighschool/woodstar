package orbit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfigResponseWireShapeMatchesOrbit(t *testing.T) {
	body, err := json.Marshal(ConfigResponse{Flags: json.RawMessage(emptyConfigFlags)})
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
