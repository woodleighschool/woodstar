package pkginfo

import (
	"encoding/json"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestSelectedMacApplicationEvidence(t *testing.T) {
	selected := plugin.Subject{Kind: "app", Path: "Tools/Editor.app", InstalledPath: "/Applications/Editor.app", App: &plugin.AppFacts{BundleID: "example.editor", Version: "2.3", Build: "203", MinimumOS: "13.0"}}
	evidence, _ := json.Marshal(selected)
	request := plugin.ReconcileRequest[json.RawMessage]{Prepared: true, Identity: plugin.Identity{Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "Editor"}}, Artifact: plugin.Artifact{Path: "leased.dmg", Format: "dmg", Evidence: map[string]json.RawMessage{"macos.application": evidence, "macos.version_key": json.RawMessage(`"CFBundleVersion"`)}}, Facts: plugin.Facts{Subjects: []plugin.Subject{selected, {Kind: "app", App: &plugin.AppFacts{BundleID: "example.other", Version: "1"}}}}}
	derived, err := Derive(request, json.RawMessage(`{}`), Derivation{})
	if err != nil {
		t.Fatal(err)
	}
	values := derived.Values
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
	derived, err = Derive(request, json.RawMessage(`{"version":"99","installs":[],"supported_architectures":["arm64"]}`), Derivation{})
	if err != nil {
		t.Fatal(err)
	}
	values = derived.Values
	if err := json.Unmarshal(raw(values), &decoded); err != nil {
		t.Fatal(err)
	}
	if values["version"] != "99" || len(decoded.Installs) != 0 || len(decoded.Architectures) != 1 || decoded.Architectures[0] != "arm64" {
		t.Fatal(values)
	}
}

func TestDerivedPackageMetadata(t *testing.T) {
	app := plugin.Subject{ID: "Suite.app", Kind: "app", InstalledPath: "/Applications/Suite.app", App: &plugin.AppFacts{BundleID: "example.suite", Version: "banana", Build: "941", MinimumOS: "15.0"}}
	request := plugin.ReconcileRequest[json.RawMessage]{Prepared: true, Identity: plugin.Identity{Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "Suite"}}, Artifact: plugin.Artifact{Path: "leased.pkg", Format: "pkg"}, Facts: plugin.Facts{Subjects: []plugin.Subject{
		{ID: ".", Kind: "container", Installer: &plugin.InstallerFacts{Version: "9.4", MinimumOS: "14.2", RestartAction: "RequireLogout"}},
		{ID: "Alpha.pkg/PackageInfo", Kind: "package", Package: &plugin.PackageFacts{Identifier: "example.alpha", Version: "4.5.6", InstalledSize: 4, HasPayload: true}},
		{ID: "Beta.pkg/PackageInfo", Kind: "package", Package: &plugin.PackageFacts{Identifier: "example.beta", Version: "7.8.9", InstalledSize: 10, HasPayload: true}},
		app,
	}}}
	derived, err := Derive(request, json.RawMessage(`{}`), Derivation{})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Version         string `json:"version"`
		MinimumOS       string `json:"minimum_os_version"`
		InstalledSize   int64  `json:"installed_size"`
		RestartAction   string `json:"RestartAction"`
		Uninstallable   bool   `json:"uninstallable"`
		UninstallMethod string `json:"uninstall_method"`
		Installs        []struct {
			VersionComparisonKey string `json:"version_comparison_key"`
			MinimumOS            string `json:"minosversion"`
		} `json:"installs"`
	}
	if err := json.Unmarshal(raw(derived.Values), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != "941" || len(decoded.Installs) != 1 || decoded.Installs[0].VersionComparisonKey != "CFBundleVersion" || decoded.Installs[0].MinimumOS != "15.0" || decoded.MinimumOS != "15.0" {
		t.Fatalf("derived detection: %s", raw(derived.Values))
	}
	if decoded.InstalledSize != 14 || decoded.RestartAction != "RequireLogout" || !decoded.Uninstallable || decoded.UninstallMethod != "removepackages" {
		t.Fatalf("derived package metadata: %s", raw(derived.Values))
	}
	request.Facts.Subjects = request.Facts.Subjects[:3]
	if derived, err = Derive(request, json.RawMessage(`{}`), Derivation{}); err != nil || derived.Values["version"] != "9.4" || derived.Values["minimum_os_version"] != "14.2" {
		t.Fatalf("installer declarations: %v, %v", derived.Values, err)
	}
	// No application remains to detect, so the derived installs list clears.
	if err := json.Unmarshal(raw(derived.Values), &decoded); err != nil || len(decoded.Installs) != 0 {
		t.Fatalf("detection without evidence: %s: %v", raw(derived.Values), err)
	}
	// Each removal field is settled on its own: the method follows the
	// installer, an item with a method is uninstallable, and one declared not
	// uninstallable has no method.
	for declared, want := range map[string][2]any{
		`{"uninstallable":true}`: {true, "removepackages"},
		`{"uninstall_method":"uninstall_script","uninstall_script":"#!/bin/sh\n"}`: {true, "uninstall_script"},
		`{"uninstallable":false}`: {false, nil},
	} {
		derived, err = Derive(request, json.RawMessage(declared), Derivation{})
		if err != nil || derived.Values["uninstallable"] != want[0] || derived.Values["uninstall_method"] != want[1] {
			t.Fatalf("%s derived removal %v %v: %v", declared, derived.Values["uninstallable"], derived.Values["uninstall_method"], err)
		}
	}
}
