//go:build postgres

package munki_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/woodleighschool/goodies/bloby"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/postgres"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestPackageUninstallPolicyRoundTripsIndependently(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "Removal Policy App"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	pkg, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID:      title.ID,
		Version:         "1.0",
		InstallerType:   packages.InstallerTypeNoPkg,
		UninstallMethod: packages.UninstallMethodRemovePackages,
		Receipts:        []packages.PackageReceipt{{PackageID: "com.example.removal-policy"}},
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.Uninstallable || pkg.UninstallMethod != packages.UninstallMethodRemovePackages {
		t.Fatalf(
			"created uninstall policy = %t/%q, want disabled with configured method",
			pkg.Uninstallable,
			pkg.UninstallMethod,
		)
	}

	pkg, err = stores.packages.Update(ctx, pkg.ID, packages.PackageMutation{
		Version:         pkg.Version,
		InstallerType:   packages.InstallerTypeNoPkg,
		Uninstallable:   true,
		UninstallMethod: packages.UninstallMethodRemovePackages,
		Receipts:        []packages.PackageReceipt{{PackageID: "com.example.removal-policy"}},
	})
	if err != nil {
		t.Fatalf("enable uninstall policy: %v", err)
	}
	if !pkg.Uninstallable || pkg.UninstallMethod != packages.UninstallMethodRemovePackages {
		t.Fatalf(
			"updated uninstall policy = %t/%q, want enabled with configured method",
			pkg.Uninstallable,
			pkg.UninstallMethod,
		)
	}
}

func TestPackageInstallerObjectValidationOwnershipAndTransitions(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	software, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "InstallerLifecycle"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	pending, _, err := stores.objects.BeginDirect(ctx, packages.ObjectPrefix, "pending.pkg")
	if err != nil {
		t.Fatalf("create pending installer: %v", err)
	}
	_, err = stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID:        software.ID,
		Version:           "pending",
		InstallerObjectID: &pending.ID,
	})
	requireErrorIs(t, "create with pending installer", err, fault.ErrInvalidInput)

	firstObject := createMunkiPackageObject(t, ctx, stores, "first.pkg", "1")
	first, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID:        software.ID,
		Version:           "1.0",
		InstallerObjectID: &firstObject.ID,
	})
	if err != nil {
		t.Fatalf("create first package: %v", err)
	}
	_, err = stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID:        software.ID,
		Version:           "owned",
		InstallerObjectID: &firstObject.ID,
	})
	requireErrorIs(t, "create with owned installer", err, fault.ErrConflict)

	secondObject := createMunkiPackageObject(t, ctx, stores, "second.pkg", "2")
	second, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID:        software.ID,
		Version:           "2.0",
		InstallerObjectID: &secondObject.ID,
	})
	if err != nil {
		t.Fatalf("create second package: %v", err)
	}
	_, err = stores.db.Exec(
		ctx,
		`UPDATE munki_packages SET installer_object_id = $1 WHERE id = $2`,
		firstObject.ID,
		second.ID,
	)
	requireErrorIs(t, "database unique owner", postgres.MutationError(err), fault.ErrAlreadyExists)

	_, err = stores.packages.Update(ctx, first.ID, packages.PackageMutation{
		Version:       first.Version,
		InstallerType: packages.InstallerTypePkg,
	})
	requireErrorIs(t, "update without explicit installer", err, fault.ErrInvalidInput)
	unchanged, err := stores.packages.GetByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("get unchanged package: %v", err)
	}
	if unchanged.InstallerObjectID == nil || *unchanged.InstallerObjectID != firstObject.ID {
		t.Fatalf("unchanged installer = %v, want %d", unchanged.InstallerObjectID, firstObject.ID)
	}

	replacement := createMunkiPackageObject(t, ctx, stores, "replacement.dmg", "3")
	replaced, err := stores.packages.Update(ctx, first.ID, packages.PackageMutation{
		Version:           first.Version,
		InstallerType:     packages.InstallerTypePkg,
		InstallerObjectID: &replacement.ID,
	})
	if err != nil {
		t.Fatalf("replace installer: %v", err)
	}
	if replaced.InstallerObjectID == nil || *replaced.InstallerObjectID != replacement.ID {
		t.Fatalf("replacement installer = %v, want %d", replaced.InstallerObjectID, replacement.ID)
	}
	assertObjectDeleted(t, ctx, stores.objects, firstObject.ID)

	packageless, err := stores.packages.Update(ctx, first.ID, packages.PackageMutation{
		Version:       first.Version,
		InstallerType: packages.InstallerTypeNoPkg,
	})
	if err != nil {
		t.Fatalf("switch to nopkg: %v", err)
	}
	if packageless.InstallerObjectID != nil {
		t.Fatalf("nopkg installer = %v, want nil", packageless.InstallerObjectID)
	}
	assertObjectDeleted(t, ctx, stores.objects, replacement.ID)

	dmg := createMunkiPackageObject(t, ctx, stores, "copy.dmg", "4")
	copied, err := stores.packages.Update(ctx, first.ID, packages.PackageMutation{
		Version:           first.Version,
		InstallerType:     packages.InstallerTypeCopyFromDMG,
		InstallerObjectID: &dmg.ID,
		ItemsToCopy: []packages.PackageItemToCopy{{
			SourceItem:      "Example.app",
			DestinationPath: "/Applications",
		}},
	})
	if err != nil {
		t.Fatalf("switch to copy_from_dmg: %v", err)
	}
	if copied.InstallerObjectID == nil || *copied.InstallerObjectID != dmg.ID {
		t.Fatalf("copy_from_dmg installer = %v, want %d", copied.InstallerObjectID, dmg.ID)
	}
}

func TestCreatePackageRejectsIconObjectAsInstaller(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "IconApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	iconObject := createMunkiIconObject(t, ctx, stores, "IconApp.png", "b")

	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:           "1.0",
			InstallerObjectID: &iconObject.ID},
	)
	if !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want invalid input", err)
	}
}

func TestPackageProjectsSoftwareIcon(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	icon := createMunkiIconObject(t, ctx, stores, "SharedApp.png", "d")
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:         "SharedIconApp",
		IconObjectID: &icon.ID,
	})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	if title.IconObjectID == nil || *title.IconObjectID != icon.ID {
		t.Fatalf("title icon object id = %v, want %d", title.IconObjectID, icon.ID)
	}
	if title.IconFile == nil ||
		title.IconFile.Filename != icon.Filename ||
		title.IconFile.SizeBytes != 512 ||
		title.IconFile.SHA256 != icon.SHA256Value() {
		t.Fatalf("title icon file = %+v, want confirmed icon metadata", title.IconFile)
	}

	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true},
	)
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.Software.IconObjectID == nil || *pkg.Software.IconObjectID != icon.ID {
		t.Fatalf("package software icon object id = %v, want %d", pkg.Software.IconObjectID, icon.ID)
	}
}

func TestPackagePreservesBlockingApplicationStates(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	stores := newMunkiStores(t, db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-blocking-apps-uuid", Serial: "C02BLOCKING"},
		OrbitNodeKey: "munki-blocking-apps-orbit",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "Blocking App"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	unset, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true},
	)
	if err != nil {
		t.Fatalf("create unset package: %v", err)
	}
	none, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:                  "2.0",
			InstallerType:            packages.InstallerTypeNoPkg,
			BlockingApplicationsNone: true,
			OnDemand:                 true},
	)
	if err != nil {
		t.Fatalf("create none package: %v", err)
	}
	populated, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:              "3.0",
			InstallerType:        packages.InstallerTypeNoPkg,
			BlockingApplications: []string{"Blocking App"},
			OnDemand:             true},
	)
	if err != nil {
		t.Fatalf("create populated package: %v", err)
	}
	_, err = stores.software.Update(ctx, title.ID, munkisoftware.UpdateMutation{
		Targets: munkisoftware.Targets{
			Include: []munkisoftware.Include{
				includeTarget(allHostsID, munkisoftware.ActionManagedInstalls),
			},
		},
	})
	if err != nil {
		t.Fatalf("target software: %v", err)
	}

	assertBlockingApplications(t, *unset, false, []string{})
	assertBlockingApplications(t, *none, true, []string{})
	assertBlockingApplications(t, *populated, false, []string{"Blocking App"})

	effective, err := stores.software.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("effective packages: %v", err)
	}
	if len(effective) != 3 {
		t.Fatalf("effective packages = %+v, want three package candidates", effective)
	}
	for _, candidate := range effective {
		switch candidate.Package.Version {
		case "1.0":
			assertBlockingApplications(t, candidate.Package, false, []string{})
		case "2.0":
			assertBlockingApplications(t, candidate.Package, true, []string{})
		case "3.0":
			assertBlockingApplications(t, candidate.Package, false, []string{"Blocking App"})
		default:
			t.Fatalf("unexpected effective package version %q", candidate.Package.Version)
		}
	}
}

func TestCreatePackageMissingRelationTargetFallsThroughToNotFound(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "MissingRelationApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	missingPackageID := int64(999999)

	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Requires: []packages.PackageReferenceMutation{
				{SoftwareID: title.ID, PackageID: missingPackageID},
			}},
	)
	if !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("CreatePackage error = %v, want ErrNotFound", err)
	}
}

func TestBulkDeletePackagesIgnoresMissingIDsAndRemovesSelectedRelations(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BulkDeletePackageApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	targetPackage := createMunkiPackage(t, ctx, stores, title.ID, title.Name, "1.0")
	dependentPackage, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:       "2.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Requires: []packages.PackageReferenceMutation{
				{SoftwareID: title.ID, PackageID: targetPackage.ID},
			}},
	)
	if err != nil {
		t.Fatalf("create dependent package: %v", err)
	}

	deleted, err := stores.packages.DeleteMany(
		ctx,
		[]int64{targetPackage.ID, dependentPackage.ID, dependentPackage.ID + 999},
	)
	if err != nil {
		t.Fatalf("bulk delete packages: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("bulk deleted = %d, want 2", deleted)
	}
	if _, err := stores.packages.GetByID(ctx, targetPackage.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("target package after bulk delete error = %v, want ErrNotFound", err)
	}
	if _, err := stores.packages.GetByID(ctx, dependentPackage.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("dependent package after bulk delete error = %v, want ErrNotFound", err)
	}
}

func TestBulkDeletePackagesReportsConflictWhileReferenced(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BulkDeleteConflictApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	targetPackage := createMunkiPackage(t, ctx, stores, title.ID, title.Name, "1.0")
	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:       "2.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Requires: []packages.PackageReferenceMutation{
				{SoftwareID: title.ID, PackageID: targetPackage.ID},
			}},
	)
	if err != nil {
		t.Fatalf("create dependent package: %v", err)
	}

	if _, err := stores.packages.DeleteMany(ctx, []int64{targetPackage.ID}); !errors.Is(err, fault.ErrConflict) {
		t.Fatalf("bulk delete referenced package error = %v, want ErrConflict", err)
	}
}

func TestDeleteObjectReportsConflictWhileReferencedByPackage(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "DeleteObjectApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	installerObject := createMunkiPackageObject(t, ctx, stores, "DeleteObject.pkg", "b")
	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:           "1.0",
			InstallerObjectID: &installerObject.ID},
	)
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	references := []struct {
		name string
		id   int64
	}{
		{name: "installer", id: installerObject.ID},
	}
	for _, ref := range references {
		if err := stores.objects.Delete(ctx, ref.id, packages.ObjectPrefix); !errors.Is(err, bloby.ErrConflict) {
			t.Fatalf("delete referenced %s object error = %v, want ErrConflict", ref.name, err)
		}
	}
	if _, err := stores.packages.DeleteMany(ctx, []int64{pkg.ID}); err != nil {
		t.Fatalf("delete package collection: %v", err)
	}
	for _, ref := range references {
		assertObjectDeleted(t, ctx, stores.objects, ref.id)
	}
}

func TestPackageStoresTypedScriptAndRelations(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "ExtraApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	dependencyTitle, err := stores.software.Create(
		ctx,
		munkisoftware.CreateMutation{Name: "DependencyApp"},
	)
	if err != nil {
		t.Fatalf("create dependency title: %v", err)
	}
	dependency := createMunkiPackage(t, ctx, stores, dependencyTitle.ID, "DependencyApp", "2.0")
	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:            "1.0",
			InstallerType:      packages.InstallerTypeNoPkg,
			InstallcheckScript: "#!/bin/zsh\nexit 0\n",
			Requires: []packages.PackageReferenceMutation{
				{SoftwareID: dependencyTitle.ID},
				{SoftwareID: dependencyTitle.ID, PackageID: dependency.ID},
			}},
	)
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.InstallcheckScript == "" || pkg.InstallerType != packages.InstallerTypeNoPkg {
		t.Fatalf("pkg typed fields = %+v, want nopkg installcheck script", pkg)
	}
	if len(pkg.Requires) != 2 {
		t.Fatalf("requires = %+v, want latest and specific dependency references", pkg.Requires)
	}
	if pkg.Requires[0].SoftwareID != dependencyTitle.ID || pkg.Requires[0].PackageID != 0 {
		t.Fatalf("first requires = %+v, want dependency software reference", pkg.Requires[0])
	}
	if pkg.Requires[0].SoftwareName != "DependencyApp" || pkg.Requires[0].PackageVersion != "" {
		t.Fatalf("first requires target = %+v, want unversioned dependency software details", pkg.Requires[0])
	}
	if pkg.Requires[1].PackageID != dependency.ID {
		t.Fatalf("second requires = %+v, want dependency package id", pkg.Requires[1])
	}
	if pkg.Requires[1].SoftwareName != "DependencyApp" || pkg.Requires[1].PackageVersion != "2.0" {
		t.Fatalf("second requires target = %+v, want dependency package details", pkg.Requires[1])
	}
}

func TestUpdatePackageReplacesEditableStateAndClearsUnusedObjects(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "Switchable App"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	installerObject := createMunkiPackageObject(t, ctx, stores, "SwitchableApp.pkg", "f")

	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:             "1.0",
			InstallerObjectID:   &installerObject.ID,
			MinimumMunkiVersion: "6.0",
			UnattendedInstall:   true,
			Requires:            []packages.PackageReferenceMutation{{SoftwareID: title.ID}}},
	)
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	updated, err := stores.packages.Update(ctx, pkg.ID, packages.PackageMutation{
		Version:            "2.0",
		InstallerType:      packages.InstallerTypeNoPkg,
		OnDemand:           true,
		InstallcheckScript: "#!/bin/sh\nexit 0\n",
	})
	if err != nil {
		t.Fatalf("update package: %v", err)
	}
	if updated.InstallerObjectID != nil {
		t.Fatalf("installer object id = %v, want cleared", updated.InstallerObjectID)
	}
	if updated.InstallerType != packages.InstallerTypeNoPkg ||
		updated.UninstallMethod != "" {
		t.Fatalf(
			"updated package modes = %s/%s, want nopkg without uninstall method",
			updated.InstallerType,
			updated.UninstallMethod,
		)
	}
	if updated.Version != "2.0" || updated.MinimumMunkiVersion != "" || updated.UnattendedInstall || len(updated.Requires) != 0 {
		t.Fatalf("updated package = %+v, want replacement package fields", updated)
	}
	assertObjectDeleted(t, ctx, stores.objects, installerObject.ID)
}

func TestPackagePatchPreservesFieldsAndReconcilesObjects(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "PatchPackage"})
	if err != nil {
		t.Fatal(err)
	}
	dependency := createMunkiPackage(t, ctx, stores, title.ID, title.Name, "1.0")
	oldObject := createMunkiPackageObject(t, ctx, stores, "old.pkg", "a")
	nextObject := createMunkiPackageObject(t, ctx, stores, "next.pkg", "b")
	deadline := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	current, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: title.ID,
		Version:    "2.0", InstallerType: packages.InstallerTypePkg, InstallerObjectID: &oldObject.ID,
		UnattendedInstall: true, SupportedArchitectures: []string{"arm64"}, Notes: "Keep notes",
		ForceInstallAfterDate: &deadline,
		Requires:              []packages.PackageReferenceMutation{{SoftwareID: title.ID, PackageID: dependency.ID}},
		PreinstallAlert:       packages.PackageAlert{Enabled: true, Title: "Keep title", Detail: "Old detail"},
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := stores.packages.Patch(ctx, current.ID, readPatch[packages.Patch](t, fmt.Sprintf(
		`{"installer_object_id":%d,"preinstall_alert":{"detail":"New detail"}}`, nextObject.ID)))
	if err != nil {
		t.Fatal(err)
	}
	want := current.Mutation()
	want.InstallerObjectID = &nextObject.ID
	want.PreinstallAlert.Detail = "New detail"
	if !reflect.DeepEqual(updated.Mutation(), want) {
		t.Fatalf("patch lost omitted fields: got %+v, want %+v", updated.Mutation(), want)
	}
	assertObjectDeleted(t, ctx, stores.objects, oldObject.ID)
	if _, err := stores.objects.GetByID(ctx, nextObject.ID); err != nil {
		t.Fatalf("new installer was removed: %v", err)
	}
	updated, err = stores.packages.Patch(ctx, current.ID, readPatch[packages.Patch](t,
		`{"installer_type":"nopkg","installer_object_id":null,"force_install_after_date":null,"notes":null,"requires":[],"supported_architectures":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if updated.InstallerObjectID != nil || updated.ForceInstallAfterDate != nil || updated.Notes != "" ||
		len(updated.Requires) != 0 || len(updated.SupportedArchitectures) != 0 || !updated.UnattendedInstall {
		t.Fatalf("explicit clear = %+v", updated)
	}
	assertObjectDeleted(t, ctx, stores.objects, nextObject.ID)
}

func TestPackagePatchRollsBackInvalidReferences(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "PatchReferences"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "OtherReferences"})
	if err != nil {
		t.Fatal(err)
	}
	otherPackage := createMunkiPackage(t, ctx, stores, other.ID, other.Name, "1.0")
	oldObject := createMunkiPackageObject(t, ctx, stores, "retained.pkg", "a")
	nextObject := createMunkiPackageObject(t, ctx, stores, "unclaimed.pkg", "b")
	icon := createMunkiIconObject(t, ctx, stores, "wrong-prefix.png", "c")
	current, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: title.ID,
		Version:    "1.0", InstallerObjectID: &oldObject.ID, Notes: "Retain", Requires: []packages.PackageReferenceMutation{{SoftwareID: other.ID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{
		fmt.Sprintf(`{"notes":"Discard","installer_object_id":%d,"requires":[{"software_id":999999}]}`, nextObject.ID),
		fmt.Sprintf(`{"notes":"Discard","requires":[{"software_id":%d,"package_id":%d}]}`, title.ID, otherPackage.ID),
		`{"installer_object_id":999999}`, `{"installer_object_id":null}`,
		fmt.Sprintf(`{"installer_object_id":%d}`, icon.ID),
	} {
		if _, err := stores.packages.Patch(ctx, current.ID, readPatch[packages.Patch](t, data)); err == nil {
			t.Fatalf("invalid patch succeeded: %s", data)
		}
		actual, err := stores.packages.GetByID(ctx, current.ID)
		if err != nil || !reflect.DeepEqual(actual.Mutation(), current.Mutation()) {
			t.Fatalf("failed patch changed current mutation: %+v, %v", actual, err)
		}
		for _, id := range []int64{oldObject.ID, nextObject.ID} {
			if _, err := stores.objects.GetByID(ctx, id); err != nil {
				t.Fatalf("failed patch deleted object %d: %v", id, err)
			}
		}
	}
	if _, err := stores.packages.Patch(ctx, 999999, packages.Patch{}); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("missing package error = %v", err)
	}
}

func TestPackagePatchPreservesConcurrentCommittedMutation(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "ConcurrentPackagePatch"})
	if err != nil {
		t.Fatal(err)
	}
	pkg := createMunkiPackage(t, ctx, stores, title.ID, title.Name, "1.0")
	document := readPatch[packages.Patch](t, `{"preinstall_alert":{"detail":"Patched detail"}}`)
	runBlockedPatch(t, db, "munki_packages", pkg.ID, func(ctx context.Context) error {
		_, err := stores.packages.Patch(ctx, pkg.ID, document)
		return err
	}, func(ctx context.Context, tx pgx.Tx) {
		if _, err := tx.Exec(ctx, `UPDATE munki_packages SET notes = 'Concurrent notes', preinstall_alert_title = 'Concurrent title' WHERE id = $1`, pkg.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO munki_package_relations (package_id, relation_kind, target_software_id, position) VALUES ($1, 'requires', $2, 0)`, pkg.ID, title.ID); err != nil {
			t.Fatal(err)
		}
	})
	actual, err := stores.packages.GetByID(ctx, pkg.ID)
	if err != nil || actual.Notes != "Concurrent notes" || actual.PreinstallAlert.Title != "Concurrent title" || actual.PreinstallAlert.Detail != "Patched detail" || len(actual.Requires) != 1 {
		t.Fatalf("concurrent package fields lost: %+v, %v", actual, err)
	}
}
