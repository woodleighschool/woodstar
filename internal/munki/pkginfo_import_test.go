package munki

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

func TestImportPkginfoPreservesSparseNativeFields(t *testing.T) {
	imported, err := ImportPkginfo(json.RawMessage(`{
		"name":"Example", "version":"1.2", "display_name":null, "unattended_install":false,
		"RestartAction":"RequireRestart", "OnDemand":false, "blocking_applications":[],
		"receipts":[{"packageid":"test.example","version":"1.2","optional":false}],
		"installs":[{"type":"application","path":"/Applications/Example.app","CFBundleIdentifier":"test.example","CFBundleShortVersionString":"1.2","CFBundleVersion":"120"}],
		"installer_choices_xml":[{"choiceIdentifier":"feature","choiceAttribute":"selected","attributeSetting":0}],
		"installer_environment":{"B":"two","A":"one"},
		"preinstall_alert":{"alert_detail":"Changed detail"}, "preuninstall_alert":null,
		"requires":["Dependency--2.0"], "update_for":[],
		"installer_item_location":"Example.pkg","installer_item_hash":"source-digest","installer_item_size":10
	}`))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := imported.Package.Apply(packages.PackageMutation{Notes: "Keep notes", UnattendedInstall: true, PreinstallAlert: packages.PackageAlert{Enabled: true, Title: "Keep title"}, PreuninstallAlert: packages.PackageAlert{Enabled: true, Title: "Preserve disabled text"}})
	if err != nil {
		t.Fatal(err)
	}
	if pkg.InstallerType != packages.InstallerTypePkg || pkg.Version != "1.2" || pkg.UnattendedInstall || pkg.OnDemand || !pkg.BlockingApplicationsNone || len(pkg.BlockingApplications) != 0 || pkg.Notes != "Keep notes" || pkg.RestartAction != packages.RestartActionRequireRestart {
		t.Fatalf("package ownership=%+v", pkg)
	}
	if pkg.Receipts[0].PackageID != "test.example" || pkg.Installs[0].BundleShortVersion != "1.2" || pkg.Installs[0].BundleVersion != "120" || pkg.InstallerChoicesXML[0].ChoiceIdentifier != "feature" || pkg.InstallerChoicesXML[0].AttributeSetting != 0 {
		t.Fatalf("native fields=%+v", pkg)
	}
	if pkg.PreinstallAlert.Title != "Keep title" || pkg.PreinstallAlert.Detail != "Changed detail" || !pkg.PreinstallAlert.Enabled || pkg.PreuninstallAlert.Enabled || pkg.PreuninstallAlert.Title != "Preserve disabled text" {
		t.Fatalf("alerts=%+v %+v", pkg.PreinstallAlert, pkg.PreuninstallAlert)
	}
	if !reflect.DeepEqual(pkg.InstallerEnvironment, []packages.PackageInstallerEnvironmentVariable{{Name: "A", Value: "one"}, {Name: "B", Value: "two"}}) {
		t.Fatalf("installer_environment=%+v", pkg.InstallerEnvironment)
	}
	if !reflect.DeepEqual(imported.Requires, []PkginfoReference{{Name: "Dependency", Version: "2.0"}}) || imported.UpdateFor == nil || len(imported.UpdateFor) != 0 || imported.InstallerItemHash != "source-digest" {
		t.Fatalf("references and content=%+v", imported)
	}
	var sparse map[string]json.RawMessage
	if err := json.Unmarshal(imported.Package.Bytes(), &sparse); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"requires", "update_for", "notes", "installer_object_id", "installer_item_hash", "installer_item_location", "installer_item_size"} {
		if _, ok := sparse[field]; ok {
			t.Errorf("unexpected managed package field %s", field)
		}
	}
}

func TestImportPkginfoRejectsUnsupportedOrInvalidNativeValues(t *testing.T) {
	for _, fields := range []string{
		`"name":null`, `"version":null`, `"version":""`, `"installer_type":null`,
		`"unknown":true`, `"restart_action":"RequireRestart"`, `"installer_object_id":42`,
		`"unattended_install":null`, `"receipts":null`, `"requires":[{"software_id":1}]`,
		`"requires":["Dependency--"]`, `"update_for":null`, `"catalogs":["production"]`,
		`"catalogs":[]`, `"icon_name":""`, `"icon_hash":""`, `"items_to_remove":[]`,
		`"icon_name":"Example.png"`, `"installer_environment":{"TEST":null}`,
		`"installer_item_hash":""`,
		`"receipts":[{"package_id":"test.example"}]`, `"installs":[{"type":"application","path":"/Example.app","bundle_identifier":"test.example"}]`,
		`"preinstall_alert":{"enabled":false}`, `"items_to_remove":[{"destination_path":"/Applications"}]`,
	} {
		t.Run(fields, func(t *testing.T) {
			if _, err := ImportPkginfo(json.RawMessage(`{"name":"Example","version":"1.0",` + fields + `}`)); err == nil {
				t.Fatal("accepted unsupported metadata")
			}
		})
	}
}

func TestImportPkginfoCopyRemovalUsesManagedCopyItems(t *testing.T) {
	_, err := ImportPkginfo(json.RawMessage(`{"name":"Example","version":"1.0","installer_type":"copy_from_dmg","uninstall_method":"remove_copied_items","items_to_copy":[{"source_item":"Example.app","destination_path":"/Applications","user":"root"}],"items_to_remove":[{"source_item":"Example.app","destination_path":"/Applications"}]}`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestImportPkginfoPreservesSoftwareClears(t *testing.T) {
	imported, err := ImportPkginfo(json.RawMessage(`{"name":"Example","version":"1.2","display_name":null}`))
	if err != nil {
		t.Fatal(err)
	}
	title, err := imported.Software.Apply(software.UpdateMutation{DisplayName: "Custom", Description: "Keep description"})
	if err != nil || title.DisplayName != "" || title.Description != "Keep description" {
		t.Fatalf("software=%+v error=%v", title, err)
	}
}
