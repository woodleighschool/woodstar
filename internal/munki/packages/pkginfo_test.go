package packages

import (
	"testing"
	"time"
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
