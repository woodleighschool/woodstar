package destination

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestPlanRejectsPaddedArtifactFilename(t *testing.T) {
	declared := metadata{name: "Example", version: "1.0", installer: plugin.Artifact{Filename: " Example.pkg", Version: "1.0", Size: 1, SHA256: strings.Repeat("a", 64)}}
	if err := json.Unmarshal(json.RawMessage(`{"version":"1.0","installer_type":"pkg"}`), &declared.pkg); err != nil {
		t.Fatal(err)
	}
	if _, err := plan(declared, Observation{}); err == nil {
		t.Fatal("accepted a filename that changes during upload")
	}
}
