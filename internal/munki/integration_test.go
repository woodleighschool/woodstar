package munki_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

type munkiStores struct {
	objects   *storage.ObjectStore
	hoststate *munki.Store
	packages  *packages.Store
	software  *munkisoftware.Store
}

func newMunkiStores(db *database.DB) munkiStores {
	objectStore := storage.NewObjectStore(db, nil)
	packageStore := packages.NewStore(db, objectStore)
	softwareStore := munkisoftware.NewStore(db, objectStore, packageStore)
	return munkiStores{
		objects:   objectStore,
		hoststate: munki.NewStore(db),
		packages:  packageStore,
		software:  softwareStore,
	}
}

// createMunkiStorageObject inserts a confirmed (available) storage object under
// prefix and returns it for use as an installer or icon in tests.
func createMunkiStorageObject(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	prefix, filename, hashChar string,
) *storage.Object {
	t.Helper()
	obj, err := stores.objects.CreatePending(ctx, prefix, filename, "application/octet-stream")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	confirmed, err := stores.objects.Confirm(ctx, obj.ID, 512, "", strings.Repeat(hashChar, 64))
	if err != nil {
		t.Fatalf("confirm object: %v", err)
	}
	return confirmed
}

func TestMunkiSoftwareIdentityIsUniqueAndSeparateFromDisplayName(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
			Eligible:      true,
		},
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.SoftwareName != "com.vendor.app" || pkg.SoftwareDisplayName == nil ||
		*pkg.SoftwareDisplayName != "Vendor App" {
		t.Fatalf(
			"package software identity = %q/%v, want canonical and presentation names",
			pkg.SoftwareName,
			pkg.SoftwareDisplayName,
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

func TestMunkiSoftwareOmitsDisplayNameWhenUnset(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	software, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "NoDisplayName"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	if software.DisplayName != nil {
		t.Fatalf("display name = %q, want nil", *software.DisplayName)
	}
}

func TestMunkiSoftwareCreateListAndResolveForHost(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	includedHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-desired-included-uuid", Serial: "C02MUNKIIN"},
		OrbitNodeKey: "munki-desired-included-orbit",
	})
	if err != nil {
		t.Fatalf("enroll included host: %v", err)
	}
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
			Eligible:      true,
		}},
	)
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	_, err = stores.software.Update(ctx, title.ID, munkisoftware.UpdateMutation{
		Category:  title.Category,
		Developer: title.Developer,
		Targets: munkisoftware.Targets{
			Include: []munkisoftware.Include{
				includeTarget(allHostsID, munkisoftware.ActionManagedInstalls),
			},
			Exclude: labelRefs([]int64{label.ID}),
		},
	})
	if err != nil {
		t.Fatalf("update software targets: %v", err)
	}

	titles, titleCount, err := stores.software.List(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list softwares: %v", err)
	}
	if titleCount != 1 || len(titles) != 1 || titles[0].Name != "GoogleChrome" {
		t.Fatalf("titles = %+v count = %d, want GoogleChrome", titles, titleCount)
	}
	packageRows, pkgCount, err := stores.packages.List(ctx, packages.PackageListParams{})
	if err != nil {
		t.Fatalf("list packages: %v", err)
	}
	if pkgCount != 1 || len(packageRows) != 1 || packageRows[0].Version != "148.0.0.1" {
		t.Fatalf("packages = %+v count = %d, want version 148.0.0.1", packageRows, pkgCount)
	}
	targets, err := stores.software.TargetsForSoftware(ctx, title.ID)
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}
	if len(targets.Include) != 1 || targets.Include[0].LabelID != allHostsID {
		t.Fatalf("targets = %+v, want one include row", targets)
	}

	included, err := stores.software.EffectivePackagesForHost(ctx, includedHost.ID)
	if err != nil {
		t.Fatalf("resolve included host: %v", err)
	}
	if len(included) != 1 || included[0].Package.SoftwareName != "GoogleChrome" ||
		!sameActions(
			included[0].Actions,
			[]munkisoftware.Action{munkisoftware.ActionManagedInstalls},
		) {
		t.Fatalf("included effective packages = %+v, want GoogleChrome install", included)
	}
	excluded, err := stores.software.EffectivePackagesForHost(ctx, excludedHost.ID)
	if err != nil {
		t.Fatalf("resolve excluded host: %v", err)
	}
	if len(excluded) != 0 {
		t.Fatalf("excluded effective packages = %+v, want none", excluded)
	}
}

func TestPackageObjectCreateListAndBindPackage(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-object-host-uuid", Serial: "C02MUNKIOBJ"},
		OrbitNodeKey: "munki-object-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "ObjectApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	installerObject := createMunkiPackageObject(t, ctx, stores, "ObjectApp.pkg", "a")

	pkg, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:           "1.0",
			InstallerObjectID: &installerObject.ID,
			Eligible:          true,
		}},
	)
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	if pkg.InstallerObjectID == nil || *pkg.InstallerObjectID != installerObject.ID {
		t.Fatalf("pkg installer object id = %v, want %d", pkg.InstallerObjectID, installerObject.ID)
	}
	replaceTargets(t, ctx, stores, title, []munkisoftware.Include{
		includeTarget(allHostsLabelID(t, ctx, labelStore), munkisoftware.ActionManagedInstalls),
	})

	effective, err := stores.software.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 ||
		effective[0].Package.InstallerObjectID == nil ||
		*effective[0].Package.InstallerObjectID != installerObject.ID {
		t.Fatalf("effective packages = %+v, want installer object id", effective)
	}
}

func TestEffectivePackagesForHostKeepsLatestCandidates(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
	if effective[0].Package.SoftwareName != "LatestApp" || effective[1].Package.SoftwareName != "LatestApp" {
		t.Fatalf("effective packages = %+v, want LatestApp candidates", effective)
	}
}

func TestCreatePackageRejectsIconObjectAsInstaller(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
			Eligible:          true,
		}},
	)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want invalid input", err)
	}
}

func TestPackageProjectsSoftwareIcon(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
			Eligible:      true,
		}},
	)
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.SoftwareIconObjectID == nil || *pkg.SoftwareIconObjectID != icon.ID {
		t.Fatalf("package software icon object id = %v, want %d", pkg.SoftwareIconObjectID, icon.ID)
	}
}

func TestRepositoryPackagesByIconObjectIDFiltersToCatalogEligiblePackages(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
			Eligible:      false,
		}},
	)
	if err != nil {
		t.Fatalf("create ineligible package: %v", err)
	}
	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "2.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Eligible:      true,
		}},
	)
	if err != nil {
		t.Fatalf("create eligible package: %v", err)
	}

	pkgs, err := stores.packages.RepositoryPackagesByIconObjectID(ctx, icon.ID)
	if err != nil {
		t.Fatalf("RepositoryPackagesByIconObjectID: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Version != "2.0" {
		t.Fatalf("icon packages = %+v, want only eligible package version 2.0", pkgs)
	}
	if pkgs[0].SoftwareIconObjectID == nil || *pkgs[0].SoftwareIconObjectID != icon.ID {
		t.Fatalf("software icon object id = %v, want %d", pkgs[0].SoftwareIconObjectID, icon.ID)
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
	db, ctx := dbtest.Open(t)
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
	if !sameActions(
		effective[0].Actions,
		[]munkisoftware.Action{munkisoftware.ActionManagedUninstalls},
	) ||
		effective[0].Package.SoftwareName != "SchoolApp" {
		t.Fatalf("effective pkg = %+v, want SAC removal of SchoolApp", effective[0])
	}
}

func TestEffectivePackagesForHostUsesRowOrderNotActionRank(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
	if !sameActions(
		effective[0].Actions,
		[]munkisoftware.Action{munkisoftware.ActionManagedInstalls},
	) ||
		effective[0].Package.Version != "1.0" {
		t.Fatalf("effective pkg = %+v, want first row install of RowOrderApp 1.0", effective[0])
	}
}

func TestCreatePackageRejectsUnsupportedArchitecture(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "Broken"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:                "1.0",
			SupportedArchitectures: []string{"ppc"},
			Eligible:               true,
		}},
	)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want invalid input", err)
	}
}

func TestPackagePreservesBlockingApplicationStates(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
			Eligible:      true,
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
			Eligible:                 true,
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
			Eligible:             true,
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
	db, ctx := dbtest.Open(t)
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
			Eligible: true,
		}},
	)
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("CreatePackage error = %v, want ErrNotFound", err)
	}
}

func TestCreatePackageRejectsInvalidRelationTarget(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "InvalidRelationApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Requires:      []packages.PackageReferenceMutation{{SoftwareID: title.ID, PackageID: -1}},
			Eligible:      true,
		}},
	)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want ErrInvalidInput", err)
	}
}

func TestDeletePackageReportsConflictWhileReferenced(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "DeletePackageApp"})
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
			Eligible: true,
		}},
	)
	if err != nil {
		t.Fatalf("create dependent package: %v", err)
	}

	if err := stores.packages.Delete(ctx, targetPackage.ID); !errors.Is(err, dbutil.ErrConflict) {
		t.Fatalf("delete referenced package error = %v, want ErrConflict", err)
	}
	if err := stores.packages.Delete(ctx, dependentPackage.ID); err != nil {
		t.Fatalf("delete dependent package: %v", err)
	}
	if _, err := stores.packages.GetByID(ctx, dependentPackage.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("dependent package after delete error = %v, want ErrNotFound", err)
	}
	if err := stores.packages.Delete(ctx, targetPackage.ID); err != nil {
		t.Fatalf("delete unreferenced package: %v", err)
	}
}

func TestBulkDeletePackagesIgnoresMissingIDsAndRemovesSelectedRelations(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
			Eligible: true,
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
	db, ctx := dbtest.Open(t)
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
			Eligible: true,
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
	db, ctx := dbtest.Open(t)
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
			Eligible:          true,
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
		if err := stores.objects.DeleteByID(ctx, ref.id); !errors.Is(err, dbutil.ErrConflict) {
			t.Fatalf("delete referenced %s object error = %v, want ErrConflict", ref.name, err)
		}
	}
	if err := stores.packages.Delete(ctx, pkg.ID); err != nil {
		t.Fatalf("delete package: %v", err)
	}
	for _, ref := range references {
		if _, err := stores.objects.GetByID(ctx, ref.id); !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("post-delete %s object lookup error = %v, want ErrNotFound", ref.name, err)
		}
	}
}

func TestCreatePackageRejectsInvalidSoftwareID(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	_, err := stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: -1, PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true,
			Eligible:      true,
		}},
	)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want ErrInvalidInput", err)
	}
}

func TestCreatePackageBadInstalledSizeFallsThroughToInvalidInput(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BadInstalledSizeApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	_, err = stores.packages.Create(
		ctx,
		packages.PackageCreateMutation{SoftwareID: title.ID, PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstalledSize: -1,
			Eligible:      true,
		}},
	)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want ErrInvalidInput", err)
	}
}

func TestPackageStoresTypedScriptAndRelations(t *testing.T) {
	db, ctx := dbtest.Open(t)
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
			Eligible: true,
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
	db, ctx := dbtest.Open(t)
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
			Eligible:          true,
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
		Eligible:           true,
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

func TestCreateTargetTargetsAllHostsLabel(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-all-devices-uuid", Serial: "C02MUNKIALL"},
		OrbitNodeKey: "munki-all-devices-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "AllDevicesApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "AllDevicesApp", "1.0")
	replaceTargets(t, ctx, stores, title, []munkisoftware.Include{
		includeTarget(allHostsLabelID(t, ctx, labelStore), munkisoftware.ActionManagedInstalls),
	})
	effective, err := stores.software.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 || effective[0].Package.SoftwareName != "AllDevicesApp" {
		t.Fatalf("effective packages = %+v, want AllDevicesApp", effective)
	}
}

func TestSoftwareTargetsRejectPinnedPackageFromAnotherSoftware(t *testing.T) {
	db, ctx := dbtest.Open(t)
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

func TestSoftwareTargetsRejectBuiltinAndIncludeOverlap(t *testing.T) {
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	title, err := stores.software.Create(
		ctx,
		munkisoftware.CreateMutation{Name: "ExcludeOverlapApp"},
	)
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	includeLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Exclude Overlap Include",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create include label: %v", err)
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
			Include: []munkisoftware.Include{
				includeTarget(includeLabel.ID, munkisoftware.ActionManagedInstalls),
			},
			Exclude: labelRefs([]int64{includeLabel.ID}),
		},
	}); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Update software overlap error = %v, want ErrInvalidInput", err)
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
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)
	labelID := allHostsLabelID(t, ctx, labelStore)

	first, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "DeletePinnedApp"})
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
	db, ctx := dbtest.Open(t)
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

func TestHostStatusUpsertAndDetail(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
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

	runStartedAt := time.Date(2026, 5, 31, 9, 23, 0, 0, time.UTC)
	runEndedAt := time.Date(2026, 5, 31, 9, 24, 14, 0, time.UTC)
	if err := stores.hoststate.UpsertHostObservation(ctx, munki.HostObservation{
		HostID:          host.ID,
		Version:         "7.1.2.5700",
		ManifestName:    "site_default",
		Success:         true,
		Errors:          []string{"first error"},
		Warnings:        []string{"first warning"},
		ProblemInstalls: []string{"Broken App"},
		RunStartedAt:    &runStartedAt,
		RunEndedAt:      &runEndedAt,
	}); err != nil {
		t.Fatalf("upsert munki host status: %v", err)
	}
	if err := stores.hoststate.ReplaceHostItems(ctx, host.ID, []munki.Item{
		{Name: "GoogleChrome", Installed: true, InstalledVersion: "148.0", RunEndedAt: &runEndedAt},
		{Name: "Optional App", Installed: false},
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
	if !detail.Success {
		t.Fatalf("success = %v, want true", detail.Success)
	}
	if len(detail.Items) != 2 || detail.Items[0].Name != "GoogleChrome" || !detail.Items[0].Installed {
		t.Fatalf("items = %+v", detail.Items)
	}
	if detail.RunStartedAt == nil || !detail.RunStartedAt.Equal(runStartedAt) ||
		detail.RunEndedAt == nil || !detail.RunEndedAt.Equal(runEndedAt) {
		t.Fatalf("detail run times = %v/%v, want stored timestamps", detail.RunStartedAt, detail.RunEndedAt)
	}
	if detail.Items[0].RunEndedAt == nil || !detail.Items[0].RunEndedAt.Equal(runEndedAt) {
		t.Fatalf("item run ended = %v, want stored timestamp", detail.Items[0].RunEndedAt)
	}
	if detail.Items[1].RunEndedAt != nil {
		t.Fatalf("optional item run ended = %v, want nil", detail.Items[1].RunEndedAt)
	}

	if err := stores.hoststate.ReplaceHostItems(
		ctx,
		host.ID,
		[]munki.Item{{Name: "Replacement", Installed: true}},
	); err != nil {
		t.Fatalf("replace munki host items again: %v", err)
	}
	detail, err = stores.hoststate.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load munki detail after replace: %v", err)
	}
	if len(detail.Items) != 1 || detail.Items[0].Name != "Replacement" {
		t.Fatalf("items after replace = %+v", detail.Items)
	}

	if err := stores.hoststate.ClearHostObservation(ctx, host.ID); err != nil {
		t.Fatalf("clear munki host status: %v", err)
	}
	if detail, err := stores.hoststate.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load cleared munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("cleared munki detail = %+v, want nil", detail)
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

func sameActions(a, b []munkisoftware.Action) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
			Eligible:      true,
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
