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
				Type:                 PackageInstallItemApplication,
				Path:                 "/Applications/Example.app",
				BundleIdentifier:     "com.example.app",
				MinimumUpdateVersion: "1.0",
			},
		},
		Receipts: []PackageReceipt{
			{PackageID: "com.example.pkg", Version: "1.2.3", Name: "Example", InstalledSize: 2048, Optional: true},
		},
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
		t.Fatalf("RestartAction = %v, want omitted when unset", got["RestartAction"])
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
	if installs[0]["type"] != "application" || installs[0]["minimum_update_version"] != "1.0" {
		t.Fatalf("installs = %#v, want required type and minimum update version", installs)
	}
	receipts := mapSlice(t, got["receipts"])
	if receipts[0][munkiReceiptPackageIDKey] != "com.example.pkg" ||
		receipts[0]["name"] != "Example" ||
		intValue(receipts[0]["installed_size"]) != 2048 ||
		receipts[0]["optional"] != true {
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

func TestPkginfoDerivesItemsToRemoveFromItemsToCopy(t *testing.T) {
	got := plistMap(t, Pkginfo(Package{
		ID:              12,
		SoftwareID:      7,
		SoftwareName:    "Example App",
		Version:         "1.2.3",
		InstallerType:   InstallerTypeCopyFromDMG,
		UninstallMethod: UninstallMethodRemoveCopiedItems,
		ItemsToCopy: []PackageItemToCopy{
			{
				SourceItem:      "Example.app",
				DestinationPath: "/Applications",
				DestinationItem: "Example.app",
				User:            "root",
				Group:           "wheel",
				Mode:            "0755",
			},
			{
				SourceItem:      "Nested/Tool.app",
				DestinationPath: "/Applications/Utilities",
			},
		},
	}, PkginfoObjects{}))

	itemsToRemove := mapSlice(t, got["items_to_remove"])
	if len(itemsToRemove) != 2 ||
		itemsToRemove[0]["destination_path"] != "/Applications" ||
		itemsToRemove[0]["destination_item"] != "Example.app" ||
		itemsToRemove[0]["source_item"] != "Example.app" {
		t.Fatalf("items_to_remove = %#v, want destination from items_to_copy", got["items_to_remove"])
	}
	if itemsToRemove[1]["destination_path"] != "/Applications/Utilities" ||
		itemsToRemove[1]["source_item"] != "Nested/Tool.app" {
		t.Fatalf(
			"items_to_remove = %#v, want source_item fallback when destination_item is absent",
			got["items_to_remove"],
		)
	}
	if _, ok := itemsToRemove[1]["destination_item"]; ok {
		t.Fatalf("items_to_remove rendered empty destination_item: %#v", itemsToRemove[1])
	}
}

func TestParseInstallerItemLocation(t *testing.T) {
	id, ok := ParseInstallerItemLocation("packages/12/installer/Example.pkg")
	if !ok || id != 12 {
		t.Fatalf("ParseInstallerItemLocation() = %d, %t, want 12, true", id, ok)
	}
	for _, value := range []string{
		"",
		"munki/packages/12/installer/Example.pkg",
		"packages/0/installer/Example.pkg",
		"packages/12/payload/Example.pkg",
		"packages/12/installer/",
		"packages/not-a-number/installer/Example.pkg",
	} {
		if id, ok := ParseInstallerItemLocation(value); ok {
			t.Fatalf("ParseInstallerItemLocation(%q) = %d, true, want false", value, id)
		}
	}
}

func TestParseIconName(t *testing.T) {
	id, ok := ParseIconName("7-Example.png")
	if !ok || id != 7 {
		t.Fatalf("ParseIconName() = %d, %t, want 7, true", id, ok)
	}
	for _, value := range []string{
		"",
		"7",
		"0-Example.png",
		"-Example.png",
		"not-a-number-Example.png",
	} {
		if id, ok := ParseIconName(value); ok {
			t.Fatalf("ParseIconName(%q) = %d, true, want false", value, id)
		}
	}
}

func TestPkginfoRendersBlockingApplicationsFromExplicitNoneSwitch(t *testing.T) {
	base := Package{
		ID:            12,
		SoftwareID:    7,
		SoftwareName:  "Example App",
		Version:       "1.2.3",
		InstallerType: InstallerTypePkg,
	}

	derived := plistMap(t, Pkginfo(base, PkginfoObjects{}))
	if _, ok := derived["blocking_applications"]; ok {
		t.Fatalf("blocking_applications rendered for derive-from-installs state: %+v", derived)
	}

	nonePackage := base
	nonePackage.BlockingApplicationsNone = true
	empty := plistMap(t, Pkginfo(nonePackage, PkginfoObjects{}))
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
