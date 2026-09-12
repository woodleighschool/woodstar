package destination

import (
	"encoding/json"
	"github.com/woodleighschool/stemma/plugin"
	"testing"
)

func TestSelectedMacApplicationEvidence(t *testing.T) {
	selected := plugin.Subject{Kind: "app", Path: "Tools/Editor.app", InstalledPath: "/Applications/Editor.app", App: &plugin.AppFacts{BundleID: "example.editor", Version: "2.3", Build: "203", MinimumOS: "13.0"}}
	evidence, _ := json.Marshal(selected)
	request := plugin.ReconcileRequest{Prepared: true, Identity: plugin.Identity{Software: "Editor"}, Artifact: plugin.Artifact{Path: "leased.dmg", Format: "dmg", Evidence: map[string]json.RawMessage{"macos.application": evidence, "macos.version_key": json.RawMessage(`"CFBundleVersion"`)}}, Facts: plugin.Facts{Subjects: []plugin.Subject{selected, {Kind: "app", App: &plugin.AppFacts{BundleID: "example.other", Version: "1"}}}}, Metadata: json.RawMessage(`{}`)}
	values, _, err := derive(request)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Installs []struct {
			BundleID   string `json:"CFBundleIdentifier"`
			Path       string `json:"path"`
			VersionKey string `json:"version_comparison_key"`
		} `json:"installs"`
		Architectures []string `json:"supported_architectures"`
	}
	if err := json.Unmarshal(raw(values), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Installs) != 1 {
		t.Fatalf("missing selected evidence: %s", raw(values))
	}
	installs := decoded.Installs[0]
	if values["version"] != "203" || installs.BundleID != "example.editor" || installs.Path != "/Applications/Editor.app" || installs.VersionKey != "CFBundleVersion" {
		t.Fatalf("native evidence: %s", raw(values))
	}
	if _, exists := values["supported_architectures"]; exists {
		t.Fatal("application evidence must not infer host eligibility")
	}
	request.Metadata = json.RawMessage(`{"pkginfo":{"version":"99","installs":[],"supported_architectures":["arm64"]}}`)
	values, _, err = derive(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw(values), &decoded); err != nil {
		t.Fatal(err)
	}
	if values["version"] != "99" || len(decoded.Installs) != 0 || len(decoded.Architectures) != 1 || decoded.Architectures[0] != "arm64" {
		t.Fatal(values)
	}
}
