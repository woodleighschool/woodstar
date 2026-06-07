package munki_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/hoststate"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

type munkiStores struct {
	artifacts      *artifacts.Store
	hoststate      *hoststate.Store
	packages       *packages.Store
	softwareTitles *munkisoftware.Store
}

func newMunkiStores(db *database.DB) munkiStores {
	artifactStore := artifacts.NewStore(db)
	packageStore := packages.NewStore(db, artifactStore)
	softwareTitleStore := munkisoftware.NewStore(db, artifactStore, packageStore)
	return munkiStores{
		artifacts:      artifactStore,
		hoststate:      hoststate.NewStore(db),
		packages:       packageStore,
		softwareTitles: softwareTitleStore,
	}
}

func TestDesiredStateCreateListAndResolveForHost(t *testing.T) {
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
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{
		Name:      "GoogleChrome",
		Category:  "Browsers",
		Developer: "Google",
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    title.ID,
		Version:       "148.0.0.1",
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
		Eligible:      true,
	})
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	_, err = stores.softwareTitles.Update(ctx, title.ID, munkisoftware.SoftwareMutation{
		Name:      title.Name,
		Category:  title.Category,
		Developer: title.Developer,
		Targets: munkisoftware.SoftwareTargets{
			Include: []munkisoftware.SoftwareInclude{
				includeTarget(allHostsID, munkisoftware.SoftwareStateManagedInstall),
			},
			Exclude: labelRefs([]int64{label.ID}),
		},
	})
	if err != nil {
		t.Fatalf("update software targets: %v", err)
	}

	titles, titleCount, err := stores.softwareTitles.List(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list software titles: %v", err)
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
	targets, err := stores.softwareTitles.TargetsForSoftwareTitle(ctx, title.ID)
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}
	if len(targets.Include) != 1 || targets.Include[0].LabelID != allHostsID {
		t.Fatalf("targets = %+v, want one include row", targets)
	}

	included, err := stores.softwareTitles.EffectivePackagesForHost(ctx, includedHost.ID)
	if err != nil {
		t.Fatalf("resolve included host: %v", err)
	}
	if len(included) != 1 || included[0].Package.SoftwareName != "GoogleChrome" ||
		included[0].State != munkisoftware.SoftwareStateManagedInstall {
		t.Fatalf("included effective packages = %+v, want GoogleChrome install", included)
	}
	excluded, err := stores.softwareTitles.EffectivePackagesForHost(ctx, excludedHost.ID)
	if err != nil {
		t.Fatalf("resolve excluded host: %v", err)
	}
	if len(excluded) != 0 {
		t.Fatalf("excluded effective packages = %+v, want none", excluded)
	}
}

func TestArtifactsCreateListAndBindPackage(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-artifact-host-uuid", Serial: "C02MUNKIART"},
		OrbitNodeKey: "munki-artifact-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "ArtifactApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	artifact, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:        artifacts.ArtifactKindPackage,
		DisplayName: "Artifact App Installer",
		Location:    "apps/ArtifactApp.pkg",
		ContentType: "application/octet-stream",
		SizeBytes:   1234,
		SHA256:      strings.Repeat("a", 64),
		StorageKey:  "pkgs/ArtifactApp.pkg",
	})
	if err != nil {
		t.Fatalf("create artifact: %v", err)
	}
	if artifact.DisplayName != "Artifact App Installer" || artifact.Location != "apps/ArtifactApp.pkg" {
		t.Fatalf("artifact = %+v", artifact)
	}
	loadedArtifact, err := stores.artifacts.GetByID(ctx, artifact.ID)
	if err != nil {
		t.Fatalf("get artifact: %v", err)
	}
	if loadedArtifact.StorageKey != "pkgs/ArtifactApp.pkg" {
		t.Fatalf("storage key = %q", loadedArtifact.StorageKey)
	}
	loadedByLocation, err := stores.artifacts.GetByLocation(ctx, artifacts.ArtifactKindPackage, "apps/ArtifactApp.pkg")
	if err != nil {
		t.Fatalf("get artifact by location: %v", err)
	}
	if loadedByLocation.ID != artifact.ID {
		t.Fatalf("artifact by location id = %d, want %d", loadedByLocation.ID, artifact.ID)
	}
	artifactRows, artifactCount, err := stores.artifacts.List(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if artifactCount != 1 || len(artifactRows) != 1 || artifactRows[0].ID != artifact.ID {
		t.Fatalf("artifacts = %+v count = %d, want created artifact", artifactRows, artifactCount)
	}

	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:          title.ID,
		Version:             "1.0",
		InstallerArtifactID: &artifact.ID,
		Eligible:            true,
	})
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	if pkg.InstallerArtifactID == nil || *pkg.InstallerArtifactID != artifact.ID {
		t.Fatalf("pkg installer artifact id = %v, want %d", pkg.InstallerArtifactID, artifact.ID)
	}
	replaceSoftwareTargets(t, ctx, stores, title, []munkisoftware.SoftwareInclude{
		includeTarget(allHostsLabelID(t, ctx, labelStore), munkisoftware.SoftwareStateManagedInstall),
	}, nil)

	effective, err := stores.softwareTitles.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 || effective[0].Package.InstallerArtifactLocation != "apps/ArtifactApp.pkg" {
		t.Fatalf("effective packages = %+v, want artifact location", effective)
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
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "LatestApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "LatestApp", "1.0")
	createMunkiPackage(t, ctx, stores, title.ID, "LatestApp", "2.0")
	replaceSoftwareTargets(t, ctx, stores, title, []munkisoftware.SoftwareInclude{
		includeTarget(allHostsLabelID(t, ctx, labelStore), munkisoftware.SoftwareStateManagedInstall),
	}, nil)

	effective, err := stores.softwareTitles.EffectivePackagesForHost(ctx, host.ID)
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

func TestCreateArtifactBadSizeFallsThroughToInvalidInput(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	_, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:       artifacts.ArtifactKindPackage,
		Location:   "apps/BadSize.pkg",
		SizeBytes:  -1,
		SHA256:     strings.Repeat("a", 64),
		StorageKey: "pkgs/BadSize.pkg",
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreateArtifact error = %v, want ErrInvalidInput", err)
	}
}

func TestCreatePackageRejectsIconArtifactAsInstaller(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "IconApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	artifact, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:       artifacts.ArtifactKindIcon,
		Location:   "IconApp.png",
		SizeBytes:  256,
		SHA256:     strings.Repeat("b", 64),
		StorageKey: "icons/IconApp.png",
	})
	if err != nil {
		t.Fatalf("create icon artifact: %v", err)
	}

	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:          title.ID,
		Version:             "1.0",
		InstallerArtifactID: &artifact.ID,
		Eligible:            true,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want invalid input", err)
	}
}

func TestPackageInheritsSoftwareIcon(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	icon := createMunkiIconArtifact(t, ctx, stores, "icons/SharedApp.png", "d")
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{
		Name:           "SharedIconApp",
		IconArtifactID: &icon.ID,
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	if title.IconArtifactID == nil || *title.IconArtifactID != icon.ID {
		t.Fatalf("title icon artifact id = %v, want %d", title.IconArtifactID, icon.ID)
	}

	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    title.ID,
		Version:       "1.0",
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
		Eligible:      true,
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.IconArtifactID != nil || pkg.IconName != "" || pkg.IconHash != "" {
		t.Fatalf(
			"package icon override = id %v name %q hash %q, want empty override",
			pkg.IconArtifactID,
			pkg.IconName,
			pkg.IconHash,
		)
	}
	if packages.EffectiveIconArtifactID(*pkg) == nil || *packages.EffectiveIconArtifactID(*pkg) != icon.ID {
		t.Fatalf("effective icon id = %v, want %d", packages.EffectiveIconArtifactID(*pkg), icon.ID)
	}
}

func TestPackageIconOverridesSoftwareIcon(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	softwareIcon := createMunkiIconArtifact(t, ctx, stores, "icons/DefaultApp.png", "e")
	packageIcon := createMunkiIconArtifact(t, ctx, stores, "icons/SpecialApp.png", "f")
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{
		Name:           "OverrideIconApp",
		IconArtifactID: &softwareIcon.ID,
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:     title.ID,
		Version:        "1.0",
		InstallerType:  packages.InstallerTypeNoPkg,
		OnDemand:       true,
		IconArtifactID: &packageIcon.ID,
		Eligible:       true,
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if packages.EffectiveIconArtifactID(*pkg) == nil || *packages.EffectiveIconArtifactID(*pkg) != packageIcon.ID {
		t.Fatalf(
			"effective icon id = %v, want package override %d",
			packages.EffectiveIconArtifactID(*pkg),
			packageIcon.ID,
		)
	}
	if pkg.IconName != "icons/SpecialApp.png" || pkg.IconHash != strings.Repeat("f", 64) {
		t.Fatalf("package icon fields = %q %q, want package override", pkg.IconName, pkg.IconHash)
	}
}

func TestUpdatePackageClearsIconOverrideToInheritSoftwareIcon(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	softwareIcon := createMunkiIconArtifact(t, ctx, stores, "icons/DefaultApp.png", "7")
	packageIcon := createMunkiIconArtifact(t, ctx, stores, "icons/SpecialApp.png", "8")
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{
		Name:           "ClearOverrideApp",
		IconArtifactID: &softwareIcon.ID,
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:     title.ID,
		Version:        "1.0",
		InstallerType:  packages.InstallerTypeNoPkg,
		OnDemand:       true,
		IconArtifactID: &packageIcon.ID,
		Eligible:       true,
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	updated, err := stores.packages.Update(ctx, pkg.ID, packages.PackageMutation{
		Version:       pkg.Version,
		InstallerType: pkg.InstallerType,
		Eligible:      pkg.Eligible,
		OnDemand:      pkg.OnDemand,
		Precache:      pkg.Precache,
	})
	if err != nil {
		t.Fatalf("update package: %v", err)
	}
	if updated.IconArtifactID != nil || updated.IconName != "" || updated.IconHash != "" {
		t.Fatalf(
			"package icon override = id %v name %q hash %q, want empty override",
			updated.IconArtifactID,
			updated.IconName,
			updated.IconHash,
		)
	}
	if packages.EffectiveIconArtifactID(*updated) == nil ||
		*packages.EffectiveIconArtifactID(*updated) != softwareIcon.ID {
		t.Fatalf(
			"effective icon id = %v, want inherited software icon %d",
			packages.EffectiveIconArtifactID(*updated),
			softwareIcon.ID,
		)
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
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "SchoolApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "SchoolApp", "1.0")

	replaceSoftwareTargets(t, ctx, stores, title, []munkisoftware.SoftwareInclude{
		includeTarget(sac.ID, munkisoftware.SoftwareStateManagedUninstall),
		includeTarget(allStudents.ID, munkisoftware.SoftwareStateManagedInstall),
	}, nil)

	effective, err := stores.softwareTitles.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 {
		t.Fatalf("effective packages = %+v, want one resolved item", effective)
	}
	if effective[0].State != munkisoftware.SoftwareStateManagedUninstall ||
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
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "RowOrderApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	installPackage := createMunkiPackage(t, ctx, stores, title.ID, "RowOrderApp", "1.0")
	removePackage := createMunkiPackage(t, ctx, stores, title.ID, "RowOrderApp", "2.0")

	replaceSoftwareTargets(t, ctx, stores, title, []munkisoftware.SoftwareInclude{
		includeSpecificTarget(installLabel.ID, munkisoftware.SoftwareStateManagedInstall, installPackage.ID),
		includeSpecificTarget(removeLabel.ID, munkisoftware.SoftwareStateManagedUninstall, removePackage.ID),
	}, nil)

	effective, err := stores.softwareTitles.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 {
		t.Fatalf("effective packages = %+v, want one resolved item", effective)
	}
	if effective[0].State != munkisoftware.SoftwareStateManagedInstall || effective[0].Package.Version != "1.0" {
		t.Fatalf("effective pkg = %+v, want first row install of RowOrderApp 1.0", effective[0])
	}
}

func TestCreatePackageRejectsUnsupportedArchitecture(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "Broken"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:             title.ID,
		Version:                "1.0",
		SupportedArchitectures: []string{"ppc"},
		Eligible:               true,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want invalid input", err)
	}
}

func TestCreatePackageMissingRelationTargetFallsThroughToNotFound(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "MissingRelationApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	missingPackageID := int64(999999)

	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    title.ID,
		Version:       "1.0",
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
		Requires:      []packages.PackageReference{{PackageID: missingPackageID}},
		Eligible:      true,
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("CreatePackage error = %v, want ErrNotFound", err)
	}
}

func TestCreatePackageRejectsInvalidRelationTarget(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "InvalidRelationApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    title.ID,
		Version:       "1.0",
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
		Requires:      []packages.PackageReference{{PackageID: -1}},
		Eligible:      true,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want ErrInvalidInput", err)
	}
}

func TestCreatePackageRejectsInvalidSoftwareID(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	_, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    -1,
		Version:       "1.0",
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
		Eligible:      true,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want ErrInvalidInput", err)
	}
}

func TestCreatePackageBadInstalledSizeFallsThroughToInvalidInput(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "BadInstalledSizeApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    title.ID,
		Version:       "1.0",
		InstalledSize: -1,
		Eligible:      true,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreatePackage error = %v, want ErrInvalidInput", err)
	}
}

func TestPackageStoresTypedScriptAndRelations(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "ExtraApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	dependencyTitle, err := stores.softwareTitles.Create(
		ctx,
		munkisoftware.SoftwareMutation{Name: "DependencyApp"},
	)
	if err != nil {
		t.Fatalf("create dependency title: %v", err)
	}
	dependency := createMunkiPackage(t, ctx, stores, dependencyTitle.ID, "DependencyApp", "2.0")
	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:         title.ID,
		Version:            "1.0",
		InstallerType:      packages.InstallerTypeNoPkg,
		InstallcheckScript: "#!/bin/zsh\nexit 0\n",
		Requires: []packages.PackageReference{
			{PackageID: dependency.ID},
		},
		Eligible: true,
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.InstallcheckScript == "" || pkg.InstallerType != packages.InstallerTypeNoPkg {
		t.Fatalf("pkg typed fields = %+v, want nopkg installcheck script", pkg)
	}
	if len(pkg.Requires) != 1 {
		t.Fatalf("requires = %+v, want dependency reference", pkg.Requires)
	}
	if pkg.Requires[0].PackageID != dependency.ID {
		t.Fatalf("first requires = %+v, want dependency package id", pkg.Requires[0])
	}
	if pkg.Requires[0].SoftwareName != "DependencyApp" || pkg.Requires[0].PackageVersion != "2.0" {
		t.Fatalf("first requires target = %+v, want dependency package details", pkg.Requires[0])
	}
}

func TestImportPackageUpsertsTypedPkginfo(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	iconArtifact, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:       artifacts.ArtifactKindIcon,
		Location:   "cccccccccccc/ImportedApp.png",
		SizeBytes:  256,
		SHA256:     strings.Repeat("c", 64),
		StorageKey: "icons/cccccccccccc/ImportedApp.png",
	})
	if err != nil {
		t.Fatalf("create icon artifact: %v", err)
	}
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{
		Name:        "Imported App",
		Description: "Managed by AutoPkg",
		Category:    "Utilities",
		Developer:   "Example",
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	dependencyTitle, err := stores.softwareTitles.Create(
		ctx,
		munkisoftware.SoftwareMutation{Name: "Python"},
	)
	if err != nil {
		t.Fatalf("create dependency title: %v", err)
	}
	dependency := createMunkiPackage(t, ctx, stores, dependencyTitle.ID, "Python", "3.12")

	pkg, err := stores.packages.Import(ctx, packages.PackageImportMutation{
		SoftwareID:     title.ID,
		IconArtifactID: &iconArtifact.ID,
		Pkginfo: fmt.Appendf(nil, `{
			"name": "ImportedApp",
			"version": "1.2.3",
			"display_name": "Imported App",
			"description": "Managed by AutoPkg",
			"category": "Utilities",
			"developer": "Example",
			"installer_type": "nopkg",
			"unattended_install": true,
			"supported_architectures": ["arm64", "x86_64"],
			"requires": ["%d"],
			"icon_name": "ImportedApp.png",
			"icon_hash": "stale",
			"installs": [{"path": "/Applications/Imported App.app"}],
			"installer_item_location": "pkgs/ImportedApp.pkg"
		}`, dependency.ID),
	})
	if err != nil {
		t.Fatalf("import package: %v", err)
	}
	if pkg.SoftwareName != "Imported App" ||
		pkg.InstallerType != packages.InstallerTypeNoPkg {
		t.Fatalf("pkg = %+v, want imported typed package", pkg)
	}
	if !pkg.UnattendedInstall || !sameStrings(pkg.SupportedArchitectures, []string{"arm64", "x86_64"}) {
		t.Fatalf("pkg typed fields = %+v", pkg)
	}
	if len(pkg.Requires) != 1 || pkg.Requires[0].PackageID != dependency.ID {
		t.Fatalf("requires = %+v, want dependency package id", pkg.Requires)
	}
	if len(pkg.Installs) != 1 || pkg.Installs[0].Path != "/Applications/Imported App.app" {
		t.Fatalf("installs = %+v, want imported install item", pkg.Installs)
	}
	if pkg.IconName != "cccccccccccc/ImportedApp.png" || pkg.IconHash != strings.Repeat("c", 64) {
		t.Fatalf("pkg icon fields = name %q hash %q, want artifact-backed icon", pkg.IconName, pkg.IconHash)
	}

	updated, err := stores.packages.Import(ctx, packages.PackageImportMutation{
		SoftwareID: title.ID,
		Pkginfo: []byte(`{
			"name": "ImportedApp",
			"version": "1.2.3",
			"display_name": "Imported App",
			"developer": "Example Updated",
			"installer_type": "nopkg",
			"OnDemand": true
		}`),
	})
	if err != nil {
		t.Fatalf("import package update: %v", err)
	}
	if updated.ID != pkg.ID {
		t.Fatalf("updated package id = %d, want upserted id %d", updated.ID, pkg.ID)
	}
	if updated.SoftwareDeveloper != "Example" {
		t.Fatalf("updated software developer = %q, want title-owned metadata", updated.SoftwareDeveloper)
	}
}

func TestImportPackageAcceptsUninstallerArtifact(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "Uninstaller App"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	installerArtifact, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:       artifacts.ArtifactKindPackage,
		Location:   "apps/UninstallerApp.pkg",
		SizeBytes:  1024,
		SHA256:     strings.Repeat("d", 64),
		StorageKey: "packages/apps/UninstallerApp.pkg",
	})
	if err != nil {
		t.Fatalf("create installer artifact: %v", err)
	}
	uninstallerArtifact, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:       artifacts.ArtifactKindPackage,
		Location:   "apps/UninstallerApp-uninstall.pkg",
		SizeBytes:  512,
		SHA256:     strings.Repeat("e", 64),
		StorageKey: "packages/apps/UninstallerApp-uninstall.pkg",
	})
	if err != nil {
		t.Fatalf("create uninstaller artifact: %v", err)
	}

	pkg, err := stores.packages.Import(ctx, packages.PackageImportMutation{
		SoftwareID:            title.ID,
		InstallerArtifactID:   &installerArtifact.ID,
		UninstallerArtifactID: &uninstallerArtifact.ID,
		Pkginfo: []byte(`{
			"name": "UninstallerApp",
			"version": "1.0",
			"uninstall_method": "uninstall_package"
		}`),
	})
	if err != nil {
		t.Fatalf("import package: %v", err)
	}
	if pkg.InstallerArtifactID == nil || *pkg.InstallerArtifactID != installerArtifact.ID {
		t.Fatalf("installer artifact id = %v, want %d", pkg.InstallerArtifactID, installerArtifact.ID)
	}
	if pkg.UninstallerArtifactID == nil || *pkg.UninstallerArtifactID != uninstallerArtifact.ID {
		t.Fatalf("uninstaller artifact id = %v, want %d", pkg.UninstallerArtifactID, uninstallerArtifact.ID)
	}
	if pkg.UninstallMethod != packages.UninstallMethodUninstallPackage {
		t.Fatalf("uninstall method = %q, want uninstall_package", pkg.UninstallMethod)
	}
}

func TestUpdatePackageReplacesEditableStateAndClearsUnusedArtifacts(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "Switchable App"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	installerArtifact, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:       artifacts.ArtifactKindPackage,
		Location:   "apps/SwitchableApp.pkg",
		SizeBytes:  1024,
		SHA256:     strings.Repeat("f", 64),
		StorageKey: "packages/apps/SwitchableApp.pkg",
	})
	if err != nil {
		t.Fatalf("create installer artifact: %v", err)
	}
	uninstallerArtifact, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:       artifacts.ArtifactKindPackage,
		Location:   "apps/SwitchableApp-uninstall.pkg",
		SizeBytes:  512,
		SHA256:     strings.Repeat("a", 64),
		StorageKey: "packages/apps/SwitchableApp-uninstall.pkg",
	})
	if err != nil {
		t.Fatalf("create uninstaller artifact: %v", err)
	}

	pkg, err := stores.packages.Import(ctx, packages.PackageImportMutation{
		SoftwareID:            title.ID,
		InstallerArtifactID:   &installerArtifact.ID,
		UninstallerArtifactID: &uninstallerArtifact.ID,
		Pkginfo: []byte(`{
			"name": "SwitchableApp",
			"version": "1.0",
			"uninstall_method": "uninstall_package"
		}`),
	})
	if err != nil {
		t.Fatalf("import package: %v", err)
	}

	updated, err := stores.packages.Update(ctx, pkg.ID, packages.PackageMutation{
		SoftwareID:         pkg.SoftwareID,
		Version:            "2.0",
		InstallerType:      packages.InstallerTypeNoPkg,
		UninstallMethod:    packages.UninstallMethodNone,
		OnDemand:           true,
		InstallcheckScript: "#!/bin/sh\nexit 0\n",
		Eligible:           true,
	})
	if err != nil {
		t.Fatalf("update package: %v", err)
	}
	if updated.InstallerArtifactID != nil {
		t.Fatalf("installer artifact id = %v, want cleared", updated.InstallerArtifactID)
	}
	if updated.UninstallerArtifactID != nil {
		t.Fatalf("uninstaller artifact id = %v, want cleared", updated.UninstallerArtifactID)
	}
	if updated.InstallerType != packages.InstallerTypeNoPkg ||
		updated.UninstallMethod != packages.UninstallMethodNone {
		t.Fatalf("updated package modes = %s/%s, want nopkg/none", updated.InstallerType, updated.UninstallMethod)
	}
	if updated.Version != "2.0" || updated.MinimumMunkiVersion != "" || len(updated.Requires) != 0 {
		t.Fatalf("updated package = %+v, want replacement state", updated)
	}
}

func TestUpdatePackageRejectsSoftwareMove(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	first, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "FirstApp"})
	if err != nil {
		t.Fatalf("create first title: %v", err)
	}
	second, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "SecondApp"})
	if err != nil {
		t.Fatalf("create second title: %v", err)
	}
	pkg := createMunkiPackage(t, ctx, stores, first.ID, "FirstApp", "1.0")

	_, err = stores.packages.Update(ctx, pkg.ID, packages.PackageMutation{
		SoftwareID:    second.ID,
		Version:       pkg.Version,
		InstallerType: pkg.InstallerType,
		Eligible:      pkg.Eligible,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("UpdatePackage error = %v, want invalid input", err)
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
	title, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "AllDevicesApp"})
	if err != nil {
		t.Fatalf("create title: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "AllDevicesApp", "1.0")
	replaceSoftwareTargets(t, ctx, stores, title, []munkisoftware.SoftwareInclude{
		includeTarget(allHostsLabelID(t, ctx, labelStore), munkisoftware.SoftwareStateManagedInstall),
	}, nil)
	effective, err := stores.softwareTitles.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 || effective[0].Package.SoftwareName != "AllDevicesApp" {
		t.Fatalf("effective packages = %+v, want AllDevicesApp", effective)
	}
}

func TestSoftwareTitleTargetsRejectPinnedPackageFromAnotherTitle(t *testing.T) {
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	first, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "FirstAssignedApp"})
	if err != nil {
		t.Fatalf("create first title: %v", err)
	}
	second, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "SecondAssignedApp"})
	if err != nil {
		t.Fatalf("create second title: %v", err)
	}
	pkg := createMunkiPackage(t, ctx, stores, first.ID, "FirstAssignedApp", "1.0")
	_, err = stores.softwareTitles.Update(ctx, second.ID, softwareTitleTargetMutation(
		second,
		[]munkisoftware.SoftwareInclude{
			includeSpecificTarget(
				allHostsLabelID(t, ctx, labelStore),
				munkisoftware.SoftwareStateManagedInstall,
				pkg.ID,
			),
		},
		nil,
	))
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Update software title target error = %v, want invalid input", err)
	}
}

func TestSoftwareTitleTargetsRejectBuiltinAndIncludeOverlap(t *testing.T) {
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(
		ctx,
		munkisoftware.SoftwareMutation{Name: "ExcludeOverlapApp"},
	)
	if err != nil {
		t.Fatalf("create title: %v", err)
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
	if _, err := stores.softwareTitles.Update(ctx, title.ID, munkisoftware.SoftwareMutation{
		Name: title.Name,
		Targets: munkisoftware.SoftwareTargets{
			Exclude: labelRefs([]int64{allHostsLabelID(t, ctx, labelStore)}),
		},
	}); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Update software title builtin exclude error = %v, want ErrInvalidInput", err)
	}
	if _, err := stores.softwareTitles.Update(ctx, title.ID, munkisoftware.SoftwareMutation{
		Name: title.Name,
		Targets: munkisoftware.SoftwareTargets{
			Include: []munkisoftware.SoftwareInclude{
				includeTarget(includeLabel.ID, munkisoftware.SoftwareStateManagedInstall),
			},
			Exclude: labelRefs([]int64{includeLabel.ID}),
		},
	}); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Update software title overlap error = %v, want ErrInvalidInput", err)
	}
	if _, err := stores.softwareTitles.Update(ctx, title.ID, munkisoftware.SoftwareMutation{
		Name: title.Name,
		Targets: munkisoftware.SoftwareTargets{
			Exclude: labelRefs([]int64{excludeLabel.ID}),
		},
	}); err != nil {
		t.Fatalf("set exclude labels: %v", err)
	}
}

func TestDeleteSoftwareTitlesCleansPackagesTargetsAndIgnoresMissingBulkIDs(t *testing.T) {
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)
	labelID := allHostsLabelID(t, ctx, labelStore)

	first, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "DeletePinnedApp"})
	if err != nil {
		t.Fatalf("create first title: %v", err)
	}
	firstPkg := createMunkiPackage(t, ctx, stores, first.ID, "DeletePinnedApp", "1.0")
	replaceSoftwareTargets(t, ctx, stores, first, []munkisoftware.SoftwareInclude{
		includeSpecificTarget(labelID, munkisoftware.SoftwareStateManagedInstall, firstPkg.ID),
	}, nil)

	if err := stores.softwareTitles.Delete(ctx, first.ID); err != nil {
		t.Fatalf("delete first title: %v", err)
	}
	if _, err := stores.softwareTitles.GetByID(ctx, first.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("GetByID after delete error = %v, want ErrNotFound", err)
	}
	assertNoMunkiChildren(t, ctx, stores, first.ID)
	if err := stores.softwareTitles.Delete(ctx, first.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("repeat delete error = %v, want ErrNotFound", err)
	}

	second, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "BulkPinnedApp"})
	if err != nil {
		t.Fatalf("create second title: %v", err)
	}
	secondPkg := createMunkiPackage(t, ctx, stores, second.ID, "BulkPinnedApp", "1.0")
	replaceSoftwareTargets(t, ctx, stores, second, []munkisoftware.SoftwareInclude{
		includeSpecificTarget(labelID, munkisoftware.SoftwareStateManagedInstall, secondPkg.ID),
	}, nil)
	third, err := stores.softwareTitles.Create(ctx, munkisoftware.SoftwareMutation{Name: "BulkPlainApp"})
	if err != nil {
		t.Fatalf("create third title: %v", err)
	}

	deleted, err := stores.softwareTitles.DeleteMany(ctx, []int64{second.ID, third.ID, third.ID + 999})
	if err != nil {
		t.Fatalf("bulk delete titles: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("bulk deleted = %d, want 2", deleted)
	}
	assertNoMunkiChildren(t, ctx, stores, second.ID)
	if _, err := stores.softwareTitles.GetByID(ctx, third.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("GetByID third after bulk delete error = %v, want ErrNotFound", err)
	}
}

func TestTargetMissingLabelFallsThroughToNotFound(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)
	title, err := stores.softwareTitles.Create(
		ctx,
		munkisoftware.SoftwareMutation{Name: "MissingLabelTarget"},
	)
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	_, err = stores.softwareTitles.Update(ctx, title.ID, softwareTitleTargetMutation(
		title,
		[]munkisoftware.SoftwareInclude{
			includeTarget(999_999, munkisoftware.SoftwareStateManagedInstall),
		},
		nil,
	))
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Update software title target error = %v, want ErrNotFound", err)
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

	success := true
	if err := stores.hoststate.UpsertHostStatus(ctx, hoststate.Observation{
		HostID:          host.ID,
		Version:         "7.1.2.5700",
		ManifestName:    "site_default",
		Success:         &success,
		Errors:          []string{"first error"},
		Warnings:        []string{"first warning"},
		ProblemInstalls: []string{"Broken App"},
		RunStartedAt:    "2026-05-31 19:23:00 +1000",
		RunEndedAt:      "2026-05-31 19:24:14 +1000",
	}); err != nil {
		t.Fatalf("upsert munki host status: %v", err)
	}
	if err := stores.hoststate.ReplaceHostItems(ctx, host.ID, []hoststate.Item{
		{Name: "GoogleChrome", Installed: true, InstalledVersion: "148.0", RunEndedAt: "2026-05-31 19:24:14 +1000"},
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
	if detail.Success == nil || !*detail.Success {
		t.Fatalf("success = %v, want true", detail.Success)
	}
	if len(detail.Items) != 2 || detail.Items[0].Name != "GoogleChrome" || !detail.Items[0].Installed {
		t.Fatalf("items = %+v", detail.Items)
	}

	if err := stores.hoststate.ReplaceHostItems(
		ctx,
		host.ID,
		[]hoststate.Item{{Name: "Replacement", Installed: true}},
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

	if err := stores.hoststate.ClearHostStatus(ctx, host.ID); err != nil {
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
	rows, _, err := labelStore.List(ctx, labels.ListParams{})
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}
	for _, row := range rows {
		if row.Name == "All Hosts" {
			return row.ID
		}
	}
	t.Fatalf("All Hosts label not found")
	return 0
}

func replaceSoftwareTargets(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	title *munkisoftware.SoftwareTitle,
	includes []munkisoftware.SoftwareInclude,
	excludeLabelIDs []int64,
) munkisoftware.SoftwareTargets {
	t.Helper()
	if _, err := stores.softwareTitles.Update(
		ctx,
		title.ID,
		softwareTitleTargetMutation(title, includes, excludeLabelIDs),
	); err != nil {
		t.Fatalf("replace software targets: %v", err)
	}
	targets, err := stores.softwareTitles.TargetsForSoftwareTitle(ctx, title.ID)
	if err != nil {
		t.Fatalf("list software targets: %v", err)
	}
	return targets
}

func softwareTitleTargetMutation(
	title *munkisoftware.SoftwareTitle,
	includes []munkisoftware.SoftwareInclude,
	excludeLabelIDs []int64,
) munkisoftware.SoftwareMutation {
	return munkisoftware.SoftwareMutation{
		Name:           title.Name,
		Description:    title.Description,
		Category:       title.Category,
		Developer:      title.Developer,
		IconArtifactID: title.IconArtifactID,
		Targets: munkisoftware.SoftwareTargets{
			Include: includes,
			Exclude: labelRefs(excludeLabelIDs),
		},
	}
}

func includeTarget(
	labelID int64,
	state munkisoftware.SoftwareDesiredState,
) munkisoftware.SoftwareInclude {
	return munkisoftware.SoftwareInclude{
		LabelID: labelID,
		Package: munkisoftware.SoftwarePackageSelector{
			Strategy: munkisoftware.SoftwarePackageLatest,
		},
		State: state,
	}
}

func includeSpecificTarget(
	labelID int64,
	state munkisoftware.SoftwareDesiredState,
	pinnedPackageID int64,
) munkisoftware.SoftwareInclude {
	return munkisoftware.SoftwareInclude{
		LabelID: labelID,
		Package: munkisoftware.SoftwarePackageSelector{
			Strategy:  munkisoftware.SoftwarePackageSpecific,
			PackageID: &pinnedPackageID,
		},
		State: state,
	}
}

func labelRefs(ids []int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(ids))
	for i, id := range ids {
		refs[i] = targeting.LabelRef{LabelID: id}
	}
	return refs
}

func sameStrings(a, b []string) bool {
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
	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    softwareID,
		Version:       version,
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
		Eligible:      true,
	})
	if err != nil {
		t.Fatalf("create pkg %s %s: %v", name, version, err)
	}
	return *pkg
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
	targets, err := stores.softwareTitles.TargetsForSoftwareTitle(ctx, softwareID)
	if err != nil {
		t.Fatalf("list targets after delete: %v", err)
	}
	if len(targets.Include) != 0 || len(targets.Exclude) != 0 {
		t.Fatalf("targets after delete = %+v, want none", targets)
	}
}

func createMunkiIconArtifact(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	location string,
	hashChar string,
) *artifacts.Artifact {
	t.Helper()
	artifact, err := stores.artifacts.Create(ctx, artifacts.ArtifactMutation{
		Kind:       artifacts.ArtifactKindIcon,
		Location:   location,
		SizeBytes:  256,
		SHA256:     strings.Repeat(hashChar, 64),
		StorageKey: location,
	})
	if err != nil {
		t.Fatalf("create icon artifact: %v", err)
	}
	return artifact
}
