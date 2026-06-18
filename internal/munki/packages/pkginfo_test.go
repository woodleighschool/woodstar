package packages

import (
	"testing"
	"time"

	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/storage"
)

const munkiReceiptPackageIDKey = "package" + "id"

func TestPkginfoProjectsMunkiTransportShape(t *testing.T) {
	forceInstallAfter := time.Date(2026, 6, 9, 10, 30, 0, 0, time.UTC)
	installerHash := "installer-sha"
	iconHash := "abc123"
	installerSize := int64(1536)
	installerID := int64(50)
	got := plistMap(t, Pkginfo(Package{
		ID:                     12,
		SoftwareID:             7,
		SoftwareName:           "Example App",
		SoftwareDescription:    "Managed by Woodstar",
		SoftwareCategory:       "Utilities",
		SoftwareDeveloper:      "Example Co",
		Version:                "1.2.3",
		InstallerType:          InstallerTypePkg,
		RestartAction:          RestartActionNone,
		MinimumMunkiVersion:    "6.0",
		SupportedArchitectures: []string{"arm64", "x86_64"},
		BlockingApplications:   []string{"Example App"},
		InstallableCondition:   "machine_type == 'laptop'",
		BlockingAppsManualQuit: true,
		BlockingAppsQuitScript: "#!/bin/sh\nosascript -e 'quit app \"Example\"'\n",
		Requires: []PackageReference{
			{SoftwareID: 8, SoftwareName: "Dependency"},
			{PackageID: 20, SoftwareID: 8, SoftwareName: "Dependency", PackageVersion: "2.0"},
		},
		UpdateFor: []PackageReference{
			{SoftwareID: 9, SoftwareName: "Updater Target"},
		},
		UnattendedInstall:     true,
		OnDemand:              true,
		ForceInstallAfterDate: &forceInstallAfter,
		InstalledSize:         42,
		InstallerEnvironment:  []PackageInstallerEnvironmentVariable{{Name: "TOKEN", Value: "value"}},
		Installs: []PackageInstallItem{
			{
				Type:             PackageInstallItemApplication,
				Path:             "/Applications/Example.app",
				BundleIdentifier: "com.example.app",
			},
		},
		Receipts:                 []PackageReceipt{{PackageID: "com.example.pkg", Version: "1.2.3", Optional: true}},
		SuppressBundleRelocation: true,
		InstallerObjectID:        &installerID,
	}, PkginfoObjects{
		Installer: &storage.Object{
			ID:        installerID,
			Prefix:    ObjectPrefix,
			Filename:  "Example.pkg",
			SizeBytes: &installerSize,
			SHA256:    &installerHash,
		},
		Icon: &storage.Object{
			ID:       7,
			Prefix:   "munki/icons",
			Filename: "Example.png",
			SHA256:   &iconHash,
		},
	}))

	if got["name"] != "7" || got["display_name"] != "Example App" || got["OnDemand"] != true {
		t.Fatalf("pkginfo identity = %+v, want Munki keys and casing", got)
	}
	if _, ok := got["installer_type"]; ok {
		t.Fatalf("installer_type = %v, want omitted for standard pkg installer", got["installer_type"])
	}
	if _, ok := got["RestartAction"]; ok {
		t.Fatalf("RestartAction = %v, want omitted when none", got["RestartAction"])
	}
	if got["icon_name"] != "7-Example.png" || got["icon_hash"] != "abc123" {
		t.Fatalf("icon fields = %v/%v, want software icon projection", got["icon_name"], got["icon_hash"])
	}
	if got["installer_item_location"] != "packages/12/installer/Example.pkg" ||
		got["installer_item_hash"] != installerHash ||
		intValue(got["installer_item_size"]) != 2 {
		t.Fatalf(
			"installer projection = %v/%v/%v, want object-derived Munki metadata",
			got["installer_item_location"],
			got["installer_item_hash"],
			got["installer_item_size"],
		)
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
	if requires, ok := stringSlice(got["requires"]); !ok ||
		len(requires) != 2 ||
		requires[0] != "8" ||
		requires[1] != "8--2.0" {
		t.Fatalf("requires = %#v, want stable unversioned and versioned Munki software IDs", got["requires"])
	}
	if updateFor, ok := stringSlice(got["update_for"]); !ok ||
		len(updateFor) != 1 ||
		updateFor[0] != "9" {
		t.Fatalf("update_for = %#v, want stable unversioned Munki software ID", got["update_for"])
	}
	installs := mapSlice(t, got["installs"])
	if installs[0]["CFBundleIdentifier"] != "com.example.app" {
		t.Fatalf("installs = %#v, want Munki bundle key", installs)
	}
	receipts := mapSlice(t, got["receipts"])
	if receipts[0][munkiReceiptPackageIDKey] != "com.example.pkg" || receipts[0]["optional"] != true {
		t.Fatalf("receipts = %#v, want Munki receipt package ID key", receipts)
	}
}

func TestPkginfoOmitsEmptyOptionalArrays(t *testing.T) {
	got := plistMap(t, Pkginfo(Package{
		ID:            12,
		SoftwareID:    7,
		SoftwareName:  "Example App",
		Version:       "1.2.3",
		InstallerType: InstallerTypePkg,
	}, PkginfoObjects{}))

	for _, key := range []string{
		"installer_choices_xml",
		"installs",
		"items_to_copy",
		"receipts",
		"requires",
		"supported_architectures",
		"update_for",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("pkginfo key %q rendered empty value in %+v", key, got)
		}
	}
}

func TestPkginfoPreservesBlockingApplicationStates(t *testing.T) {
	base := Package{
		ID:            12,
		SoftwareID:    7,
		SoftwareName:  "Example App",
		Version:       "1.2.3",
		InstallerType: InstallerTypePkg,
	}

	omitted := plistMap(t, Pkginfo(base, PkginfoObjects{}))
	if _, ok := omitted["blocking_applications"]; ok {
		t.Fatalf("blocking_applications rendered for nil state: %+v", omitted)
	}

	emptyPackage := base
	emptyPackage.BlockingApplications = []string{}
	empty := plistMap(t, Pkginfo(emptyPackage, PkginfoObjects{}))
	if values, ok := stringSlice(empty["blocking_applications"]); !ok || len(values) != 0 {
		t.Fatalf("blocking_applications = %#v, want explicit empty list", empty["blocking_applications"])
	}

	populatedPackage := base
	populatedPackage.BlockingApplications = []string{"Example App"}
	populated := plistMap(t, Pkginfo(populatedPackage, PkginfoObjects{}))
	if values, ok := stringSlice(populated["blocking_applications"]); !ok ||
		len(values) != 1 ||
		values[0] != "Example App" {
		t.Fatalf("blocking_applications = %#v, want populated list", populated["blocking_applications"])
	}
}

func plistMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := plist.Marshal(value, plist.XMLFormat)
	if err != nil {
		t.Fatalf("marshal pkginfo plist: %v", err)
	}
	var out map[string]any
	if _, err := plist.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal pkginfo plist: %v", err)
	}
	return out
}

func mapSlice(t *testing.T, value any) []map[string]any {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("value = %#v, want plist array", value)
	}
	out := make([]map[string]any, len(items))
	for i, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("item %d = %#v, want plist dict", i, item)
		}
		out[i] = row
	}
	return out
}

func stringSlice(value any) ([]string, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, len(items))
	for i, item := range items {
		value, ok := item.(string)
		if !ok {
			return nil, false
		}
		out[i] = value
	}
	return out, true
}

func intValue(value any) int64 {
	switch value := value.(type) {
	case int64:
		return value
	case uint64:
		return int64(value)
	case int:
		return int64(value)
	default:
		return 0
	}
}
