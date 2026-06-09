package packages

import (
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const munkiReceiptPackageIDKey = "package" + "id"

func TestPkginfoProjectsMunkiTransportShape(t *testing.T) {
	forceInstallAfter := time.Date(2026, 6, 9, 10, 30, 0, 0, time.UTC)
	got := Pkginfo(Package{
		ID:                     12,
		SoftwareName:           "Example App",
		SoftwareDescription:    "Managed by Woodstar",
		SoftwareCategory:       "Utilities",
		SoftwareDeveloper:      "Example Co",
		Version:                "1.2.3",
		InstallerType:          InstallerTypePkg,
		RestartAction:          RestartActionNone,
		MinimumMunkiVersion:    "6.0",
		SupportedArchitectures: []string{"arm64", "x86_64"},
		BlockingApplications:   []string{},
		InstallableCondition:   "machine_type == 'laptop'",
		BlockingAppsManualQuit: true,
		BlockingAppsQuitScript: "#!/bin/sh\nosascript -e 'quit app \"Example\"'\n",
		Requires:               []PackageReference{{PackageID: 20}},
		UpdateFor:              []PackageReference{{PackageID: 21}},
		UnattendedInstall:      true,
		OnDemand:               true,
		ForceInstallAfterDate:  &forceInstallAfter,
		InstalledSize:          42,
		InstallerEnvironment:   []PackageInstallerEnvironmentVariable{{Name: "TOKEN", Value: "value"}},
		Installs: []PackageInstallItem{
			{
				Type:             PackageInstallItemApplication,
				Path:             "/Applications/Example.app",
				BundleIdentifier: "com.example.app",
			},
		},
		Receipts:                 []PackageReceipt{{PackageID: "com.example.pkg", Version: "1.2.3", Optional: true}},
		SuppressBundleRelocation: true,
	}, IconRef{Name: "icons/Example.png", Hash: "abc123"})

	if got["name"] != "12" || got["display_name"] != "Example App" || got["OnDemand"] != true {
		t.Fatalf("pkginfo identity = %+v, want Munki keys and casing", got)
	}
	if _, ok := got["installer_type"]; ok {
		t.Fatalf("installer_type = %v, want omitted for standard pkg installer", got["installer_type"])
	}
	if _, ok := got["RestartAction"]; ok {
		t.Fatalf("RestartAction = %v, want omitted when none", got["RestartAction"])
	}
	if got["icon_name"] != "icons/Example.png" || got["icon_hash"] != "abc123" {
		t.Fatalf("icon fields = %v/%v, want software icon projection", got["icon_name"], got["icon_hash"])
	}
	if got["force_install_after_date"] != forceInstallAfter {
		t.Fatalf("force_install_after_date = %#v, want time.Time", got["force_install_after_date"])
	}
	if got["installable_condition"] != "machine_type == 'laptop'" {
		t.Fatalf("installable_condition = %#v, want Munki predicate", got["installable_condition"])
	}
	if got["blocking_applications_manual_quit_only"] != true || got["blocking_applications_quit_script"] == "" {
		t.Fatalf(
			"blocking app handling = %#v/%#v, want Munki 7 keys",
			got["blocking_applications_manual_quit_only"],
			got["blocking_applications_quit_script"],
		)
	}
	if requires := got["requires"].([]string); len(requires) != 1 || requires[0] != "20" {
		t.Fatalf("requires = %#v, want Woodstar package ID string", got["requires"])
	}
	installs := got["installs"].([]map[string]any)
	if installs[0]["CFBundleIdentifier"] != "com.example.app" {
		t.Fatalf("installs = %#v, want Munki bundle key", installs)
	}
	receipts := got["receipts"].([]map[string]any)
	if receipts[0][munkiReceiptPackageIDKey] != "com.example.pkg" || receipts[0]["optional"] != true {
		t.Fatalf("receipts = %#v, want Munki receipt package ID key", receipts)
	}
}

func TestPackageMutationFromPkginfoParsesTypedBoundary(t *testing.T) {
	mutation, err := packageMutationFromPkginfo([]byte(`{
		"name": "ExampleApp",
		"version": "1.2.3",
		"installer_type": "nopkg",
		"RestartAction": "RequireRestart",
		"OnDemand": true,
		"installable_condition": "machine_type == 'laptop'",
		"blocking_applications_manual_quit_only": true,
		"blocking_applications_quit_script": "#!/bin/sh\nexit 0\n",
		"requires": ["20"],
		"update_for": ["21"],
		"installer_environment": {"TOKEN": "value"},
		"installs": [{"path": "/Applications/Example.app", "CFBundleIdentifier": "com.example.app"}],
		"receipts": [{"` + munkiReceiptPackageIDKey + `": "com.example.pkg", "optional": true}],
		"items_to_copy": [{"source_item": "Example.app", "destination_path": "/Applications"}],
		"preinstall_alert": {"alert_title": "Heads up", "ok_label": "Install"}
	}`))
	if err != nil {
		t.Fatalf("packageMutationFromPkginfo: %v", err)
	}
	if mutation.Version != "1.2.3" || mutation.InstallerType != InstallerTypeNoPkg || !mutation.OnDemand {
		t.Fatalf("mutation identity = %+v, want typed pkginfo fields", mutation)
	}
	if mutation.RestartAction != RestartActionRequireRestart {
		t.Fatalf("restart action = %q, want RequireRestart", mutation.RestartAction)
	}
	if mutation.InstallableCondition != "machine_type == 'laptop'" || !mutation.BlockingAppsManualQuit ||
		mutation.BlockingAppsQuitScript == "" {
		t.Fatalf(
			"Munki 7 fields = %+v/%t/%q, want typed package fields",
			mutation.InstallableCondition,
			mutation.BlockingAppsManualQuit,
			mutation.BlockingAppsQuitScript,
		)
	}
	if len(mutation.Requires) != 1 || mutation.Requires[0].PackageID != 20 {
		t.Fatalf("requires = %+v, want Woodstar package reference", mutation.Requires)
	}
	if len(mutation.UpdateFor) != 1 || mutation.UpdateFor[0].PackageID != 21 {
		t.Fatalf("update_for = %+v, want Woodstar package reference", mutation.UpdateFor)
	}
	if len(mutation.InstallerEnvironment) != 1 || mutation.InstallerEnvironment[0].Name != "TOKEN" {
		t.Fatalf("installer environment = %+v, want typed environment", mutation.InstallerEnvironment)
	}
	if len(mutation.Installs) != 1 || mutation.Installs[0].Type != PackageInstallItemFile ||
		mutation.Installs[0].BundleIdentifier != "com.example.app" {
		t.Fatalf("installs = %+v, want default file type and bundle fields", mutation.Installs)
	}
	if len(mutation.Receipts) != 1 || mutation.Receipts[0].PackageID != "com.example.pkg" ||
		!mutation.Receipts[0].Optional {
		t.Fatalf("receipts = %+v, want Munki receipt package ID parsed", mutation.Receipts)
	}
	if !mutation.PreinstallAlert.Enabled || mutation.PreinstallAlert.Title != "Heads up" {
		t.Fatalf("preinstall alert = %+v, want enabled alert", mutation.PreinstallAlert)
	}
}

func TestPackageMutationFromPkginfoRejectsWrongShape(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "non object", raw: []byte(`[]`)},
		{name: "wrong scalar type", raw: []byte(`{"version": 12}`)},
		{name: "wrong nested type", raw: []byte(`{"version": "1.0", "installs": [{"path": 12}]}`)},
		{name: "bad reference", raw: []byte(`{"version": "1.0", "requires": ["latest"]}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := packageMutationFromPkginfo(tt.raw)
			if !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("packageMutationFromPkginfo error = %v, want ErrInvalidInput", err)
			}
		})
	}
}
