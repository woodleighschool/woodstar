package destination

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestValidationDoesNotContactDestination(t *testing.T) {
	request := plugin.ReconcileRequest{Method: "validate", Config: raw(map[string]any{"url": "https://woodstar.test", "api_key": "synthetic-key"})}
	// Validation never contacts the destination, so label names are only checked for shape.
	request.Metadata = json.RawMessage(`{"targets":{"include":[{"label_name":"Staff","actions":["optional_installs","managed_updates"]}],"exclude":[{"label_name":"Loaners"}]}}`)
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	for _, metadata := range []string{
		`null`, `{"software":{}}`, `{"package":{}}`,
		`{"targets":null}`, `{"targets":[]}`, `{"targets":{"include":null}}`, `{"targets":{"exclude":null}}`, `{"targets":{"unknown":true}}`,
		// Native label ids and package pins cannot be declared.
		`{"targets":{"include":[{"label_id":7,"actions":["managed_installs"]}]}}`,
		`{"targets":{"include":[{"label_name":"Staff","package":{"strategy":"latest"},"actions":["managed_installs"]}]}}`,
		`{"targets":{"exclude":[{"label_id":9}]}}`,
		`{"targets":{"include":[{"label_name":"Staff"}]}}`,
		`{"targets":{"include":[{"label_name":"Staff","actions":["install"]}]}}`,
		`{"targets":{"exclude":[{"label_name":" "}]}}`,
	} {
		t.Run(metadata, func(t *testing.T) {
			request.Metadata = json.RawMessage(metadata)
			if _, err := Handle(t.Context(), request); err == nil {
				t.Fatal("accepted invalid controls")
			}
		})
	}
}

func TestDerivedFieldsFollowTheCurrentEvidence(t *testing.T) {
	fixture, connection := serveFixture(t)
	derive := map[string]any{"app": map[string]any{"subject": "app"}}
	request := plugin.ReconcileRequest{
		Method: "apply", Prepared: true, Config: connection,
		Identity: plugin.Identity{Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "Example"}},
		Subjects: map[string]plugin.SubjectSelector{"app": {Kind: "app", Path: "Example.app"}},
		Facts:    plugin.Facts{Subjects: []plugin.Subject{{Kind: "app", Path: "Example.app", App: &plugin.AppFacts{BundleID: "test.example", Version: "1.0", Build: "100", MinimumOS: "14.0"}}}},
		Metadata: raw(map[string]any{"derive": derive}),
		Artifact: installerFixture(t, "Example.dmg", "first installer", "1.0"),
	}
	apply := func() map[string]any {
		t.Helper()
		if _, err := Handle(t.Context(), request); err != nil {
			t.Fatal(err)
		}
		return fixture.version("1.0")
	}
	pkg := apply()
	installs, _ := pkg["installs"].([]any)
	if len(installs) != 1 {
		t.Fatalf("installs=%v", installs)
	}
	app, _ := installs[0].(map[string]any)
	if app["path"] != "/Applications/Example.app" || app["bundle_identifier"] != "test.example" || app["bundle_version"] != "100" || pkg["minimum_os_version"] != "14.0" {
		t.Fatalf("derived package=%v", pkg)
	}

	// No run remembers what an earlier one derived, so a field derivation owns
	// is cleared as soon as the evidence stops supplying it.
	request.Facts.Subjects[0].App.MinimumOS = ""
	if pkg = apply(); pkg["minimum_os_version"] != nil || fixture.uploadCount() != 1 || fixture.packageCount() != 1 {
		t.Fatalf("withdrawn evidence: package=%v uploads=%d packages=%d", pkg, fixture.uploadCount(), fixture.packageCount())
	}
	assertFixtureConverged(t, fixture, request)
	request.Facts.Subjects[0].App.MinimumOS = "14.0"
	request.Metadata = raw(map[string]any{"derive": derive, "pkginfo": map[string]any{"minimum_os_version": "13.0"}})
	if pkg = apply(); pkg["minimum_os_version"] != "13.0" {
		t.Fatalf("declared value lost to evidence: %v", pkg["minimum_os_version"])
	}
	assertFixtureConverged(t, fixture, request)

	request.Metadata, request.Subjects = nil, nil
	request.Artifact = installerFixture(t, "Example.pkg", "package installer", "1.0")
	request.Facts.Subjects = []plugin.Subject{
		{Kind: "container", Installer: &plugin.InstallerFacts{RestartAction: "RequireRestart"}},
		{Kind: "package", Package: &plugin.PackageFacts{Identifier: "test.example", Version: "1.0", InstalledSize: 128, HasPayload: true}},
	}
	pkg = apply()
	copies, _ := pkg["items_to_copy"].([]any)
	receipts, _ := pkg["receipts"].([]any)
	if len(copies) != 0 || len(receipts) != 1 || number(pkg["installed_size"]) != 128 || pkg["restart_action"] != "RequireRestart" {
		t.Fatalf("DMG to PKG metadata: %v", pkg)
	}
	request.Artifact = installerFixture(t, "Example.dmg", "disk image installer", "1.0")
	request.Facts.Subjects = []plugin.Subject{{Kind: "app", Path: "Example.app", App: &plugin.AppFacts{BundleID: "test.example", Version: "1.0"}}}
	pkg = apply()
	copies, _ = pkg["items_to_copy"].([]any)
	receipts, _ = pkg["receipts"].([]any)
	if len(copies) != 1 || len(receipts) != 0 || number(pkg["installed_size"]) != 0 || pkg["restart_action"] != nil {
		t.Fatalf("PKG to DMG metadata: %v", pkg)
	}
	assertFixtureConverged(t, fixture, request)
}

func TestDerivationClearsOnlyTheFieldsItOwns(t *testing.T) {
	for _, test := range []struct {
		name, metadata string
		want           []string
	}{
		{"no evidence", `{}`, []string{"package.installed_size", "package.installs", "package.minimum_os_version", "package.receipts", "package.restart_action", "package.uninstall_method", "package.uninstallable"}},
		// Declared receipts are the detection, and removal still derives from them.
		{"declared detection", `{"pkginfo":{"receipts":[{"packageid":"example.app","version":"1"}]}}`, []string{"package.installed_size", "package.minimum_os_version", "package.restart_action"}},
		{"declared removal", `{"pkginfo":{"uninstallable":true,"uninstall_method":"uninstall_script","uninstall_script":"#!/bin/sh\nexit 0"}}`, []string{"package.installed_size", "package.installs", "package.minimum_os_version", "package.receipts", "package.restart_action", "package.uninstall_method", "package.uninstall_script"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, connection := serveFixture(t)
			installer := installerFixture(t, "Example.pkg", "synthetic installer bytes", "1.0")
			fixture.software = map[string]any{"id": json.Number("1"), "name": "Example", "targets": map[string]any{"include": []any{}, "exclude": []any{}}}
			fixture.objects[40], fixture.names[40] = []byte("synthetic installer bytes"), installer.Filename
			// Every field a PKG's derivation owns holds a value, alongside two it never supplies.
			fixture.setPackage(map[string]any{
				"id": json.Number("5"), "version": "1.0", "installer_type": "pkg", "installer_object_id": json.Number("40"),
				"receipts":       []any{map[string]any{"package_id": "example.app", "version": "1"}},
				"installed_size": 2048, "restart_action": "RequireRestart", "minimum_os_version": "13.0",
				"installs":      []any{map[string]any{"type": "application", "path": "/Applications/Example.app", "bundle_identifier": "example.app", "bundle_short_version": "1"}},
				"uninstallable": true, "uninstall_method": "removepackages",
				"notes": "Operator note", "unattended_install": true,
			})
			response, err := Handle(t.Context(), plugin.ReconcileRequest{Method: "plan", Prepared: true, Config: connection, Identity: plugin.Identity{Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "Example"}}, Metadata: json.RawMessage(test.metadata), Artifact: installer})
			if err != nil {
				t.Fatal(err)
			}
			if got := changedFields(response.Changes); !slices.Equal(got, test.want) || fixture.writes() != 0 {
				t.Fatalf("planned=%v, want %v (writes=%d)", got, test.want, fixture.writes())
			}
		})
	}
}
