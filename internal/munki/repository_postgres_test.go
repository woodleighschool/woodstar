//go:build postgres

package munki_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

type munkiStores struct {
	db        *database.DB
	objects   *storage.ObjectStore
	hoststate *munki.Store
	packages  *packages.Store
	software  *munkisoftware.Store
}

func newMunkiStores(db *database.DB) munkiStores {
	objectStore := storage.NewObjectStore(db, nil, slog.New(slog.DiscardHandler))
	packageStore := packages.NewStore(db, objectStore)
	softwareStore := munkisoftware.NewStore(db, objectStore, packageStore)
	return munkiStores{
		db:        db,
		objects:   objectStore,
		hoststate: munki.NewStore(db),
		packages:  packageStore,
		software:  softwareStore,
	}
}

// createMunkiStorageObject inserts an available storage object under
// prefix and returns it for use as an installer or icon in tests.
func createMunkiStorageObject(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	prefix, filename, hashChar string,
) *storage.Object {
	t.Helper()
	contentType := "application/octet-stream"
	if prefix == munkisoftware.IconObjectPrefix {
		contentType = "image/png"
	}
	var objectID int64
	err := stores.db.Pool().QueryRow(ctx, `
INSERT INTO storage_objects (
    prefix, filename, content_type, size_bytes, sha256, available_at
) VALUES ($1, $2, $3, 512, $4, now())
RETURNING id`, prefix, filename, contentType, strings.Repeat(hashChar, 64)).Scan(&objectID)
	if err != nil {
		t.Fatalf("insert available object: %v", err)
	}
	object, err := stores.objects.GetByID(ctx, objectID)
	if err != nil {
		t.Fatalf("get available object: %v", err)
	}
	return object
}

func TestMunkiSoftwareIdentityIsUniqueAndSeparateFromDisplayName(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)

	software, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:        "com.vendor.app",
		DisplayName: "Vendor App",
	})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	if software.Name != "com.vendor.app" || software.DisplayName == nil || *software.DisplayName != "Vendor App" {
		t.Fatalf(
			"software identity = %q/%v, want canonical and presentation names",
			software.Name,
			software.DisplayName,
		)
	}

	pkg, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: software.ID,
		PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
		},
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.Software.Name != "com.vendor.app" || pkg.Software.DisplayName == nil ||
		*pkg.Software.DisplayName != "Vendor App" {
		t.Fatalf(
			"package software identity = %q/%v, want canonical and presentation names",
			pkg.Software.Name,
			pkg.Software.DisplayName,
		)
	}
	packageRows, count, err := stores.packages.List(ctx, packages.PackageListParams{
		ListParams: dbutil.ListParams{Q: "Vendor App"},
	})
	if err != nil {
		t.Fatalf("search packages by visible software name: %v", err)
	}
	if count != 1 || len(packageRows) != 1 || packageRows[0].ID != pkg.ID {
		t.Fatalf("visible-name package search = %+v count %d, want package %d", packageRows, count, pkg.ID)
	}

	_, err = stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:        "com.vendor.app",
		DisplayName: "Duplicate Vendor App",
	})
	if !errors.Is(err, dbutil.ErrAlreadyExists) {
		t.Fatalf("duplicate canonical name error = %v, want already exists", err)
	}
}

func TestPackageUninstallPolicyRoundTripsIndependently(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "Removal Policy App"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	pkg, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: title.ID,
		PackageMutation: packages.PackageMutation{
			Version:         "1.0",
			InstallerType:   packages.InstallerTypeNoPkg,
			UninstallMethod: packages.UninstallMethodRemovePackages,
			Receipts:        []packages.PackageReceipt{{PackageID: "com.example.removal-policy"}},
		},
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

func TestMunkiSoftwareExclusionOverridesAllHostsInclude(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	excludedHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-desired-excluded-uuid", Serial: "C02MUNKIOUT"},
		OrbitNodeKey: "munki-desired-excluded-orbit",
	})
	if err != nil {
		t.Fatalf("enroll excluded host: %v", err)
	}
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Munki Desired Test",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{excludedHost.ID},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:      "GoogleChrome",
		Category:  "Browsers",
		Developer: "Google",
	})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "148.0.0.1",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
		}},
	)
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	include := []munkisoftware.Include{
		includeTarget(allHostsID, munkisoftware.ActionManagedInstalls),
	}
	_, err = stores.software.Update(ctx, title.ID, munkisoftware.UpdateMutation{
		Category:  title.Category,
		Developer: title.Developer,
		Targets: munkisoftware.Targets{
			Include: include,
		},
	})
	if err != nil {
		t.Fatalf("include all hosts: %v", err)
	}
	effective, err := stores.software.EffectivePackagesForHost(ctx, excludedHost.ID)
	if err != nil {
		t.Fatalf("resolve host before exclusion: %v", err)
	}
	if len(effective) != 1 || effective[0].Package.Software.Name != "GoogleChrome" {
		t.Fatalf("effective packages before exclusion = %+v, want GoogleChrome", effective)
	}

	_, err = stores.software.Update(ctx, title.ID, munkisoftware.UpdateMutation{
		Category:  title.Category,
		Developer: title.Developer,
		Targets: munkisoftware.Targets{
			Include: include,
			Exclude: labelRefs([]int64{label.ID}),
		},
	})
	if err != nil {
		t.Fatalf("add software exclusion: %v", err)
	}
	excluded, err := stores.software.EffectivePackagesForHost(ctx, excludedHost.ID)
	if err != nil {
		t.Fatalf("resolve excluded host: %v", err)
	}
	if len(excluded) != 0 {
		t.Fatalf("excluded effective packages = %+v, want none", excluded)
	}
}

func TestPackageInstallerObjectValidationOwnershipAndTransitions(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)
	software, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "InstallerLifecycle"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	pending, err := stores.objects.CreatePending(ctx, packages.ObjectPrefix, "pending.pkg")
	if err != nil {
		t.Fatalf("create pending installer: %v", err)
	}
	_, err = stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: software.ID,
		PackageMutation: packages.PackageMutation{
			Version:           "pending",
			InstallerObjectID: &pending.ID,
		},
	})
	requireErrorIs(t, "create with pending installer", err, dbutil.ErrInvalidInput)

	firstObject := createMunkiPackageObject(t, ctx, stores, "first.pkg", "1")
	first, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: software.ID,
		PackageMutation: packages.PackageMutation{
			Version:           "1.0",
			InstallerObjectID: &firstObject.ID,
		},
	})
	if err != nil {
		t.Fatalf("create first package: %v", err)
	}
	_, err = stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: software.ID,
		PackageMutation: packages.PackageMutation{
			Version:           "owned",
			InstallerObjectID: &firstObject.ID,
		},
	})
	requireErrorIs(t, "create with owned installer", err, dbutil.ErrConflict)

	secondObject := createMunkiPackageObject(t, ctx, stores, "second.pkg", "2")
	second, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: software.ID,
		PackageMutation: packages.PackageMutation{
			Version:           "2.0",
			InstallerObjectID: &secondObject.ID,
		},
	})
	if err != nil {
		t.Fatalf("create second package: %v", err)
	}
	_, err = stores.db.Pool().Exec(
		ctx,
		`UPDATE munki_packages SET installer_object_id = $1 WHERE id = $2`,
		firstObject.ID,
		second.ID,
	)
	requireErrorIs(t, "database unique owner", dbutil.MutationError(err), dbutil.ErrAlreadyExists)

	_, err = stores.packages.Update(ctx, first.ID, packages.PackageMutation{
		Version:       first.Version,
		InstallerType: packages.InstallerTypePkg,
	})
	requireErrorIs(t, "update without explicit installer", err, dbutil.ErrInvalidInput)
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

func requireErrorIs(t *testing.T, operation string, err error, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("%s error = %v, want %v", operation, err, target)
	}
}

func TestEffectivePackagesForHostKeepsLatestCandidates(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-latest-host-uuid", Serial: "C02MUNKILATEST"},
		OrbitNodeKey: "munki-latest-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "LatestApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "LatestApp", "1.0")
	createMunkiPackage(t, ctx, stores, title.ID, "LatestApp", "2.0")
	replaceTargets(t, ctx, stores, title, []munkisoftware.Include{
		includeTarget(allHostsLabelID(t, ctx, labelStore), munkisoftware.ActionManagedInstalls),
	})

	effective, err := stores.software.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 2 {
		t.Fatalf("effective packages = %+v, want two latest candidates", effective)
	}
	if effective[0].Package.Software.Name != "LatestApp" || effective[1].Package.Software.Name != "LatestApp" {
		t.Fatalf("effective packages = %+v, want LatestApp candidates", effective)
	}
}

func TestCreatePackageRejectsIconObjectAsInstaller(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "IconApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	iconObject := createMunkiIconObject(t, ctx, stores, "IconApp.png", "b")

	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:           "1.0",
			InstallerObjectID: &iconObject.ID,
		}},
	)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want invalid input", err)
	}
}

func TestPackageProjectsSoftwareIcon(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)

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
		title.IconFile.SHA256 != strings.Repeat("d", 64) {
		t.Fatalf("title icon file = %+v, want confirmed icon metadata", title.IconFile)
	}

	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
		}},
	)
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.Software.IconObjectID == nil || *pkg.Software.IconObjectID != icon.ID {
		t.Fatalf("package software icon object id = %v, want %d", pkg.Software.IconObjectID, icon.ID)
	}
}

func TestRepositoryPackagesByIconObjectIDIncludesEveryPackage(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)

	icon := createMunkiIconObject(t, ctx, stores, "CatalogIconApp.png", "e")
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:         "CatalogIconApp",
		IconObjectID: &icon.ID,
	})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
		}},
	)
	if err != nil {
		t.Fatalf("create first package: %v", err)
	}
	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "2.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
		}},
	)
	if err != nil {
		t.Fatalf("create second package: %v", err)
	}

	pkgs, err := stores.packages.RepositoryPackagesByIconObjectID(ctx, icon.ID)
	if err != nil {
		t.Fatalf("RepositoryPackagesByIconObjectID: %v", err)
	}
	if len(pkgs) != 2 || pkgs[0].Version != "1.0" || pkgs[1].Version != "2.0" {
		t.Fatalf("icon packages = %+v, want both package versions", pkgs)
	}
	if pkgs[0].Software.IconObjectID == nil || *pkgs[0].Software.IconObjectID != icon.ID {
		t.Fatalf("software icon object id = %v, want %d", pkgs[0].Software.IconObjectID, icon.ID)
	}
	iconIDs, err := stores.packages.ListRepositoryIconObjectIDs(ctx)
	if err != nil {
		t.Fatalf("ListRepositoryIconObjectIDs: %v", err)
	}
	if len(iconIDs) != 1 || iconIDs[0] != icon.ID {
		t.Fatalf("repository icon object IDs = %v, want [%d]", iconIDs, icon.ID)
	}
}

func TestEffectivePackagesForHostUsesPriorityForSchoolTargets(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-sac-student-uuid", Serial: "C02MUNKISAC"},
		OrbitNodeKey: "munki-sac-student-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	allStudents, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Azure - All Students",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{host.ID},
	})
	if err != nil {
		t.Fatalf("create all students label: %v", err)
	}
	sac, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "SAC",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{host.ID},
	})
	if err != nil {
		t.Fatalf("create SAC label: %v", err)
	}
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "SchoolApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "SchoolApp", "1.0")

	replaceTargets(t, ctx, stores, title, []munkisoftware.Include{
		includeTarget(sac.ID, munkisoftware.ActionManagedUninstalls),
		includeTarget(allStudents.ID, munkisoftware.ActionManagedInstalls),
	})

	effective, err := stores.software.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 {
		t.Fatalf("effective packages = %+v, want one resolved item", effective)
	}
	if !slices.Equal(
		effective[0].Actions,
		[]munkisoftware.Action{munkisoftware.ActionManagedUninstalls},
	) ||
		effective[0].Package.Software.Name != "SchoolApp" {
		t.Fatalf("effective pkg = %+v, want SAC removal of SchoolApp", effective[0])
	}
}

func TestEffectivePackagesForHostUsesRowOrderNotActionRank(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-row-order-uuid", Serial: "C02MUNKIRO"},
		OrbitNodeKey: "munki-row-order-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	installLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Munki Row Order Install",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{host.ID},
	})
	if err != nil {
		t.Fatalf("create install label: %v", err)
	}
	removeLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Munki Row Order Remove",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{host.ID},
	})
	if err != nil {
		t.Fatalf("create remove label: %v", err)
	}
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "RowOrderApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	installPackage := createMunkiPackage(t, ctx, stores, title.ID, "RowOrderApp", "1.0")
	removePackage := createMunkiPackage(t, ctx, stores, title.ID, "RowOrderApp", "2.0")

	replaceTargets(t, ctx, stores, title, []munkisoftware.Include{
		includeSpecificTarget(installLabel.ID, munkisoftware.ActionManagedInstalls, installPackage.ID),
		includeSpecificTarget(removeLabel.ID, munkisoftware.ActionManagedUninstalls, removePackage.ID),
	})

	effective, err := stores.software.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 {
		t.Fatalf("effective packages = %+v, want one resolved item", effective)
	}
	if !slices.Equal(
		effective[0].Actions,
		[]munkisoftware.Action{munkisoftware.ActionManagedInstalls},
	) ||
		effective[0].Package.Version != "1.0" {
		t.Fatalf("effective pkg = %+v, want first row install of RowOrderApp 1.0", effective[0])
	}
}

func TestPackagePreservesBlockingApplicationStates(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-blocking-apps-uuid", Serial: "C02BLOCKING"},
		OrbitNodeKey: "munki-blocking-apps-orbit",
	})
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
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
		}},
	)
	if err != nil {
		t.Fatalf("create unset package: %v", err)
	}
	none, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:                  "2.0",
			InstallerType:            packages.InstallerTypeNoPkg,
			BlockingApplicationsNone: true,
			OnDemand:                 true,
		}},
	)
	if err != nil {
		t.Fatalf("create none package: %v", err)
	}
	populated, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:              "3.0",
			InstallerType:        packages.InstallerTypeNoPkg,
			BlockingApplications: []string{"Blocking App"},
			OnDemand:             true,
		}},
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
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "MissingRelationApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	missingPackageID := int64(999999)

	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Requires: []packages.PackageReferenceMutation{
				{SoftwareID: title.ID, PackageID: missingPackageID},
			},
		}},
	)
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("CreatePackage error = %v, want ErrNotFound", err)
	}
}

func TestBulkDeletePackagesIgnoresMissingIDsAndRemovesSelectedRelations(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BulkDeletePackageApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	targetPackage := createMunkiPackage(t, ctx, stores, title.ID, title.Name, "1.0")
	dependentPackage, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "2.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Requires: []packages.PackageReferenceMutation{
				{SoftwareID: title.ID, PackageID: targetPackage.ID},
			},
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
	if _, err := stores.packages.GetByID(ctx, targetPackage.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("target package after bulk delete error = %v, want ErrNotFound", err)
	}
	if _, err := stores.packages.GetByID(ctx, dependentPackage.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("dependent package after bulk delete error = %v, want ErrNotFound", err)
	}
}

func TestBulkDeletePackagesReportsConflictWhileReferenced(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BulkDeleteConflictApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	targetPackage := createMunkiPackage(t, ctx, stores, title.ID, title.Name, "1.0")
	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "2.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Requires: []packages.PackageReferenceMutation{
				{SoftwareID: title.ID, PackageID: targetPackage.ID},
			},
		}},
	)
	if err != nil {
		t.Fatalf("create dependent package: %v", err)
	}

	if _, err := stores.packages.DeleteMany(ctx, []int64{targetPackage.ID}); !errors.Is(err, dbutil.ErrConflict) {
		t.Fatalf("bulk delete referenced package error = %v, want ErrConflict", err)
	}
}

func TestDeleteObjectReportsConflictWhileReferencedByPackage(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "DeleteObjectApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	installerObject := createMunkiPackageObject(t, ctx, stores, "DeleteObject.pkg", "b")
	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:           "1.0",
			InstallerObjectID: &installerObject.ID,
		}},
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
		if err := stores.objects.Delete(ctx, ref.id); !errors.Is(err, dbutil.ErrConflict) {
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
	stores := newMunkiStores(db)

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
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:            "1.0",
			InstallerType:      packages.InstallerTypeNoPkg,
			InstallcheckScript: "#!/bin/zsh\nexit 0\n",
			Requires: []packages.PackageReferenceMutation{
				{SoftwareID: dependencyTitle.ID},
				{SoftwareID: dependencyTitle.ID, PackageID: dependency.ID},
			},
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
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "Switchable App"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	installerObject := createMunkiPackageObject(t, ctx, stores, "SwitchableApp.pkg", "f")

	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:           "1.0",
			InstallerObjectID: &installerObject.ID,
		}},
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
	if updated.Version != "2.0" || updated.MinimumMunkiVersion != "" || len(updated.Requires) != 0 {
		t.Fatalf("updated package = %+v, want replacement package fields", updated)
	}
}

func TestSoftwareTargetsRejectPinnedPackageFromAnotherSoftware(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	first, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "FirstAssignedApp"})
	if err != nil {
		t.Fatalf("create first software: %v", err)
	}
	second, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "SecondAssignedApp"})
	if err != nil {
		t.Fatalf("create second software: %v", err)
	}
	pkg := createMunkiPackage(t, ctx, stores, first.ID, "FirstAssignedApp", "1.0")
	_, err = stores.software.Update(ctx, second.ID, softwareTargetMutation(
		second,
		[]munkisoftware.Include{
			includeSpecificTarget(
				allHostsLabelID(t, ctx, labelStore),
				munkisoftware.ActionManagedInstalls,
				pkg.ID,
			),
		},
		nil,
	))
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Update software target error = %v, want invalid input", err)
	}
}

func TestSoftwareTargetsRejectBuiltinExclude(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(
		ctx,
		munkisoftware.CreateMutation{Name: "ExcludeOverlapApp"},
	)
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	excludeLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Exclude Overlap Exclude",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create exclude label: %v", err)
	}
	if _, err := stores.software.Update(ctx, title.ID, munkisoftware.UpdateMutation{
		Targets: munkisoftware.Targets{
			Exclude: labelRefs([]int64{allHostsLabelID(t, ctx, labelStore)}),
		},
	}); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Update software builtin exclude error = %v, want ErrInvalidInput", err)
	}
	if _, err := stores.software.Update(ctx, title.ID, munkisoftware.UpdateMutation{
		Targets: munkisoftware.Targets{
			Exclude: labelRefs([]int64{excludeLabel.ID}),
		},
	}); err != nil {
		t.Fatalf("set exclude labels: %v", err)
	}
}

func TestDeleteMunkiSoftwareCleansPackagesTargetsAndIgnoresMissingBulkIDs(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)
	labelID := allHostsLabelID(t, ctx, labelStore)

	firstIcon := createMunkiIconObject(t, ctx, stores, "DeletePinnedApp.png", "f")
	first, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:         "DeletePinnedApp",
		IconObjectID: &firstIcon.ID,
	})
	if err != nil {
		t.Fatalf("create first software: %v", err)
	}
	firstPkg := createMunkiPackage(t, ctx, stores, first.ID, "DeletePinnedApp", "1.0")
	replaceTargets(t, ctx, stores, first, []munkisoftware.Include{
		includeSpecificTarget(labelID, munkisoftware.ActionManagedInstalls, firstPkg.ID),
	})

	if err := stores.software.Delete(ctx, first.ID); err != nil {
		t.Fatalf("delete first software: %v", err)
	}
	if _, err := stores.software.GetByID(ctx, first.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("GetByID after delete error = %v, want ErrNotFound", err)
	}
	assertObjectDeleted(t, ctx, stores.objects, firstIcon.ID)
	assertNoMunkiChildren(t, ctx, stores, first.ID)
	if err := stores.software.Delete(ctx, first.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("repeat delete error = %v, want ErrNotFound", err)
	}

	second, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BulkPinnedApp"})
	if err != nil {
		t.Fatalf("create second software: %v", err)
	}
	secondPkg := createMunkiPackage(t, ctx, stores, second.ID, "BulkPinnedApp", "1.0")
	replaceTargets(t, ctx, stores, second, []munkisoftware.Include{
		includeSpecificTarget(labelID, munkisoftware.ActionManagedInstalls, secondPkg.ID),
	})
	third, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BulkPlainApp"})
	if err != nil {
		t.Fatalf("create third software: %v", err)
	}

	deleted, err := stores.software.DeleteMany(ctx, []int64{second.ID, third.ID, third.ID + 999})
	if err != nil {
		t.Fatalf("bulk delete software: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("bulk deleted = %d, want 2", deleted)
	}
	assertNoMunkiChildren(t, ctx, stores, second.ID)
	if _, err := stores.software.GetByID(ctx, third.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("GetByID third after bulk delete error = %v, want ErrNotFound", err)
	}
}

func TestTargetMissingLabelFallsThroughToNotFound(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(db)
	title, err := stores.software.Create(
		ctx,
		munkisoftware.CreateMutation{Name: "MissingLabelTarget"},
	)
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	_, err = stores.software.Update(ctx, title.ID, softwareTargetMutation(
		title,
		[]munkisoftware.Include{
			includeTarget(999_999, munkisoftware.ActionManagedInstalls),
		},
		nil,
	))
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Update software target error = %v, want ErrNotFound", err)
	}
}

func TestHostMunkiStateKeepsDesiredSoftwareSeparateFromExactObservations(t *testing.T) { //nolint:cyclop,funlen // One desired/observed database lifecycle.
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-host-observation-uuid", Serial: "C02MUNKI"},
		OrbitNodeKey: "munki-host-observation-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	if detail, err := stores.hoststate.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load absent munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("absent munki detail = %+v, want nil", detail)
	}

	allHostsID := allHostsLabelID(t, ctx, labelStore)
	vscodeIcon := createMunkiStorageObject(
		t,
		ctx,
		stores,
		munkisoftware.IconObjectPrefix,
		"VisualStudioCode.png",
		"b",
	)
	vscode, err := stores.software.Create(
		ctx,
		munkisoftware.CreateMutation{
			Name:         "VisualStudioCode",
			IconObjectID: &vscodeIcon.ID,
		},
	)
	if err != nil {
		t.Fatalf("create VisualStudioCode: %v", err)
	}
	vscodePackage := createMunkiPackage(t, ctx, stores, vscode.ID, vscode.Name, "1.130.0")
	replaceTargets(t, ctx, stores, vscode, []munkisoftware.Include{
		includeSpecificTarget(
			allHostsID,
			munkisoftware.ActionManagedUpdates,
			vscodePackage.ID,
		),
	})
	chrome, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "GoogleChrome"})
	if err != nil {
		t.Fatalf("create GoogleChrome: %v", err)
	}
	createMunkiPackage(t, ctx, stores, chrome.ID, chrome.Name, "148.0")
	replaceTargets(t, ctx, stores, chrome, []munkisoftware.Include{
		includeTarget(allHostsID, munkisoftware.ActionManagedInstalls),
	})

	runStartedAt := time.Date(2026, 5, 31, 9, 23, 0, 0, time.UTC)
	runEndedAt := time.Date(2026, 5, 31, 9, 24, 14, 0, time.UTC)
	if err := stores.hoststate.UpsertHostObservation(ctx, munki.HostObservation{
		HostID:          host.ID,
		Version:         "7.1.2.5700",
		ManifestName:    "site_default",
		Errors:          []string{"first error"},
		Warnings:        []string{"first warning"},
		ProblemInstalls: []string{"Broken App"},
		RunStartedAt:    &runStartedAt,
		RunEndedAt:      &runEndedAt,
	}); err != nil {
		t.Fatalf("upsert munki host status: %v", err)
	}
	if err := stores.hoststate.ReplaceHostItems(ctx, host.ID, []munki.ItemObservation{
		{
			Name:          "NotGoogleChrome",
			DisplayName:   "GoogleChrome",
			TargetVersion: "148.0",
		},
		{
			Name:             "VisualStudioCode",
			DisplayName:      "Visual Studio Code",
			Installed:        false,
			InstalledVersion: "",
			TargetVersion:    "1.130.0",
		},
	}); err != nil {
		t.Fatalf("replace munki host items: %v", err)
	}

	detail, err := stores.hoststate.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load munki detail: %v", err)
	}
	if detail == nil {
		t.Fatal("munki detail is nil")
	}
	if detail.Version != "7.1.2.5700" || detail.ManifestName != "site_default" {
		t.Fatalf("detail = %+v, want version and manifest", detail)
	}
	if detail.RunStartedAt == nil || !detail.RunStartedAt.Equal(runStartedAt) ||
		detail.RunEndedAt == nil || !detail.RunEndedAt.Equal(runEndedAt) {
		t.Fatalf("detail run times = %v/%v, want stored timestamps", detail.RunStartedAt, detail.RunEndedAt)
	}

	desired, count, err := stores.software.ListForHost(
		ctx,
		host.ID,
		munkisoftware.HostManifestSoftwareListParams{},
	)
	if err != nil {
		t.Fatalf("list host Munki software: %v", err)
	}
	if count != 2 || len(desired) != 2 {
		t.Fatalf("desired software = %+v count %d, want two manifest items", desired, count)
	}
	if desired[0].Software.Name != "GoogleChrome" || desired[0].Observation != nil {
		t.Fatalf(
			"GoogleChrome = %+v, want no observation from matching display_name",
			desired[0],
		)
	}
	vscodeState := desired[1]
	if vscodeState.Software.Name != "VisualStudioCode" ||
		vscodeState.Software.IconURL != munkisoftware.IconURL(&vscodeIcon.ID) ||
		!slices.Equal(vscodeState.Actions, []munkisoftware.Action{munkisoftware.ActionManagedUpdates}) ||
		vscodeState.Package.Strategy != munkisoftware.PackageSpecific ||
		vscodeState.Package.ID == nil ||
		*vscodeState.Package.ID != vscodePackage.ID ||
		vscodeState.Package.Version != "1.130.0" ||
		vscodeState.Observation == nil ||
		vscodeState.Observation.DisplayName != "Visual Studio Code" ||
		vscodeState.Observation.Installed ||
		vscodeState.Observation.InstalledVersion != "" ||
		vscodeState.Observation.TargetVersion != "1.130.0" {
		t.Fatalf("VisualStudioCode = %+v, want exact pending update observation", vscodeState)
	}

	nextVSCodePackage := createMunkiPackage(t, ctx, stores, vscode.ID, vscode.Name, "1.131.0")
	replaceTargets(t, ctx, stores, vscode, []munkisoftware.Include{
		includeSpecificTarget(
			allHostsID,
			munkisoftware.ActionManagedUpdates,
			nextVSCodePackage.ID,
		),
	})
	if err := stores.hoststate.ReplaceHostItems(ctx, host.ID, []munki.ItemObservation{{
		Name:             "VisualStudioCode",
		DisplayName:      "Visual Studio Code",
		Installed:        true,
		InstalledVersion: "1.130.0",
	}}); err != nil {
		t.Fatalf("replace Munki items with satisfied prior target: %v", err)
	}
	desired, _, err = stores.software.ListForHost(
		ctx,
		host.ID,
		munkisoftware.HostManifestSoftwareListParams{},
	)
	if err != nil {
		t.Fatalf("list host Munki software after retargeting: %v", err)
	}
	vscodeState = desired[1]
	if vscodeState.Package.ID == nil ||
		*vscodeState.Package.ID != nextVSCodePackage.ID ||
		vscodeState.Package.Version != "1.131.0" ||
		vscodeState.Observation == nil ||
		!vscodeState.Observation.Installed ||
		vscodeState.Observation.InstalledVersion != "1.130.0" {
		t.Fatalf(
			"VisualStudioCode after retargeting = %+v, want new desired target and unchanged prior observation",
			vscodeState,
		)
	}

	if err := stores.hoststate.ClearHostObservation(ctx, host.ID); err != nil {
		t.Fatalf("clear munki host status: %v", err)
	}
	if detail, err := stores.hoststate.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load cleared munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("cleared munki detail = %+v, want nil", detail)
	}
	desired, _, err = stores.software.ListForHost(
		ctx,
		host.ID,
		munkisoftware.HostManifestSoftwareListParams{},
	)
	if err != nil {
		t.Fatalf("list desired software after clearing observations: %v", err)
	}
	if desired[1].Observation != nil {
		t.Fatalf("VisualStudioCode after clear = %+v, want desired row without observation", desired[1])
	}
}

func allHostsLabelID(t *testing.T, ctx context.Context, labelStore *labels.Store) int64 {
	t.Helper()
	rows, _, err := labelStore.List(ctx, labels.LabelListParams{})
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}
	for _, row := range rows {
		if row.BuiltinKey != nil && *row.BuiltinKey == labels.BuiltinKeyAllHosts {
			return row.ID
		}
	}
	t.Fatalf("All Hosts label not found")
	return 0
}

func replaceTargets(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	title *munkisoftware.Software,
	includes []munkisoftware.Include,
) munkisoftware.Targets {
	t.Helper()
	if _, err := stores.software.Update(
		ctx,
		title.ID,
		softwareTargetMutation(title, includes, nil),
	); err != nil {
		t.Fatalf("replace software targets: %v", err)
	}
	targets, err := stores.software.TargetsForSoftware(ctx, title.ID)
	if err != nil {
		t.Fatalf("list software targets: %v", err)
	}
	return targets
}

func softwareTargetMutation(
	title *munkisoftware.Software,
	includes []munkisoftware.Include,
	excludeLabelIDs []int64,
) munkisoftware.UpdateMutation {
	displayName := ""
	if title.DisplayName != nil {
		displayName = *title.DisplayName
	}
	return munkisoftware.UpdateMutation{
		DisplayName:  displayName,
		Description:  title.Description,
		Category:     title.Category,
		Developer:    title.Developer,
		IconObjectID: title.IconObjectID,
		Targets: munkisoftware.Targets{
			Include: includes,
			Exclude: labelRefs(excludeLabelIDs),
		},
	}
}

func includeTarget(
	labelID int64,
	actions ...munkisoftware.Action,
) munkisoftware.Include {
	return munkisoftware.Include{
		LabelID: labelID,
		Package: munkisoftware.PackageSelector{
			Strategy: munkisoftware.PackageLatest,
		},
		Actions: actions,
	}
}

func includeSpecificTarget(
	labelID int64,
	action munkisoftware.Action,
	pinnedPackageID int64,
) munkisoftware.Include {
	return munkisoftware.Include{
		LabelID: labelID,
		Package: munkisoftware.PackageSelector{
			Strategy:  munkisoftware.PackageSpecific,
			PackageID: &pinnedPackageID,
		},
		Actions: []munkisoftware.Action{action},
	}
}

func labelRefs(ids []int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(ids))
	for i, id := range ids {
		refs[i] = targeting.LabelRef{LabelID: id}
	}
	return refs
}

func createMunkiPackage(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	softwareID int64,
	name string,
	version string,
) packages.Package {
	t.Helper()
	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: softwareID, PackageMutation: packages.PackageMutation{
			Version:       version,
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
		}},
	)
	if err != nil {
		t.Fatalf("create pkg %s %s: %v", name, version, err)
	}
	return *pkg
}

func createMunkiPackageObject(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	location string,
	hashChar string,
) *storage.Object {
	t.Helper()
	return createMunkiStorageObject(t, ctx, stores, "munki/packages", location, hashChar)
}

func assertBlockingApplications(t *testing.T, pkg packages.Package, wantNone bool, want []string) {
	t.Helper()
	if pkg.BlockingApplicationsNone != wantNone {
		t.Fatalf(
			"package %s blocking_applications_none = %t, want %t",
			pkg.Version,
			pkg.BlockingApplicationsNone,
			wantNone,
		)
	}
	if len(pkg.BlockingApplications) != len(want) {
		t.Fatalf("package %s blocking applications = %#v, want %#v", pkg.Version, pkg.BlockingApplications, want)
	}
	for i := range want {
		if pkg.BlockingApplications[i] != want[i] {
			t.Fatalf("package %s blocking applications = %#v, want %#v", pkg.Version, pkg.BlockingApplications, want)
		}
	}
}

func assertNoMunkiChildren(t *testing.T, ctx context.Context, stores munkiStores, softwareID int64) {
	t.Helper()
	pkgRows, pkgCount, err := stores.packages.List(ctx, packages.PackageListParams{SoftwareID: softwareID})
	if err != nil {
		t.Fatalf("list packages after delete: %v", err)
	}
	if pkgCount != 0 || len(pkgRows) != 0 {
		t.Fatalf("packages after delete = %+v count = %d, want none", pkgRows, pkgCount)
	}
	targets, err := stores.software.TargetsForSoftware(ctx, softwareID)
	if err != nil {
		t.Fatalf("list targets after delete: %v", err)
	}
	if len(targets.Include) != 0 || len(targets.Exclude) != 0 {
		t.Fatalf("targets after delete = %+v, want none", targets)
	}
}

func assertObjectDeleted(
	t *testing.T,
	ctx context.Context,
	objects *storage.ObjectStore,
	objectID int64,
) {
	t.Helper()
	if _, err := objects.GetByID(ctx, objectID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get deleted object %d error = %v, want ErrNotFound", objectID, err)
	}
}

func createMunkiIconObject(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	location string,
	hashChar string,
) *storage.Object {
	t.Helper()
	return createMunkiStorageObject(t, ctx, stores, "munki/icons", location, hashChar)
}
