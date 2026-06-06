package munki_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/assignments"
	"github.com/woodleighschool/woodstar/internal/munki/hoststate"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/softwaretitles"
)

type munkiStores struct {
	artifacts      *artifacts.Store
	assignments    *assignments.Store
	hoststate      *hoststate.Store
	packages       *packages.Store
	softwareTitles *softwaretitles.Store
}

func newMunkiStores(db *database.DB) munkiStores {
	artifactStore := artifacts.NewStore(db)
	softwareTitleStore := softwaretitles.NewStore(db, artifactStore)
	packageStore := packages.NewStore(db, softwareTitleStore, artifactStore)
	assignmentStore := assignments.NewStore(db, softwareTitleStore, packageStore)
	return munkiStores{
		artifacts:      artifactStore,
		assignments:    assignmentStore,
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{
		Name:        "GoogleChrome",
		DisplayName: "Google Chrome",
		Category:    "Browsers",
		Developer:   "Google",
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    title.ID,
		Name:          "GoogleChrome",
		Version:       "148.0.0.1",
		InstallerType: packages.InstallerTypeNoPkg,
		Eligible:      true,
	})
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	excludeLabelIDs, err := stores.assignments.ReplaceExcludeLabelIDs(ctx, title.ID, []int64{label.ID})
	if err != nil {
		t.Fatalf("set exclude labels: %v", err)
	}
	if len(excludeLabelIDs) != 1 || excludeLabelIDs[0] != label.ID {
		t.Fatalf("exclude labels = %v, want [%d]", excludeLabelIDs, label.ID)
	}
	include, err := stores.assignments.Create(ctx, includeAssignment(
		title.ID,
		1,
		allHostsID,
		assignments.AssignmentActionInstall,
	))
	if err != nil {
		t.Fatalf("create include assignment: %v", err)
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
	assignmentRows, assignmentCount, err := stores.assignments.List(ctx, assignments.AssignmentListParams{})
	if err != nil {
		t.Fatalf("list assignments: %v", err)
	}
	if assignmentCount != 1 || len(assignmentRows) != 1 || assignmentRows[0].ID != include.ID {
		t.Fatalf("assignments = %+v count = %d, want one include row", assignmentRows, assignmentCount)
	}

	included, err := stores.assignments.EffectivePackagesForHost(ctx, includedHost.ID)
	if err != nil {
		t.Fatalf("resolve included host: %v", err)
	}
	if len(included) != 1 || included[0].Package.Name != "GoogleChrome" ||
		included[0].Action != assignments.AssignmentActionInstall {
		t.Fatalf("included effective packages = %+v, want GoogleChrome install", included)
	}
	excluded, err := stores.assignments.EffectivePackagesForHost(ctx, excludedHost.ID)
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "ArtifactApp"})
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
		Name:                "ArtifactApp",
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
	if _, err := stores.assignments.Create(ctx, includeAssignment(
		title.ID,
		1,
		allHostsLabelID(t, ctx, labelStore),
		assignments.AssignmentActionInstall,
	)); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	effective, err := stores.assignments.EffectivePackagesForHost(ctx, host.ID)
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "LatestApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "LatestApp", "1.0")
	createMunkiPackage(t, ctx, stores, title.ID, "LatestApp", "2.0")
	if _, err := stores.assignments.Create(ctx, includeAssignment(
		title.ID,
		1,
		allHostsLabelID(t, ctx, labelStore),
		assignments.AssignmentActionInstall,
	)); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	effective, err := stores.assignments.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 2 {
		t.Fatalf("effective packages = %+v, want two latest candidates", effective)
	}
	if effective[0].Package.Name != "LatestApp" || effective[1].Package.Name != "LatestApp" {
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

	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "IconApp"})
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
		Name:                "IconApp",
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{
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
		SoftwareID: title.ID,
		Name:       "SharedIconApp",
		Version:    "1.0",
		Eligible:   true,
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{
		Name:           "OverrideIconApp",
		IconArtifactID: &softwareIcon.ID,
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:     title.ID,
		Name:           "OverrideIconApp",
		Version:        "1.0",
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{
		Name:           "ClearOverrideApp",
		IconArtifactID: &softwareIcon.ID,
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:     title.ID,
		Name:           "ClearOverrideApp",
		Version:        "1.0",
		IconArtifactID: &packageIcon.ID,
		Eligible:       true,
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	updated, err := stores.packages.Update(ctx, pkg.ID, packages.PackageMutation{
		Name:     pkg.Name,
		Version:  pkg.Version,
		Eligible: pkg.Eligible,
		OnDemand: pkg.OnDemand,
		Precache: pkg.Precache,
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

func TestEffectivePackagesForHostUsesPriorityForSchoolAssignments(t *testing.T) {
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "SchoolApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "SchoolApp", "1.0")

	_, err = stores.assignments.Create(ctx, includeAssignment(title.ID, 1, sac.ID, assignments.AssignmentActionRemove))
	if err != nil {
		t.Fatalf("create SAC remove assignment: %v", err)
	}
	_, err = stores.assignments.Create(
		ctx,
		includeAssignment(title.ID, 2, allStudents.ID, assignments.AssignmentActionInstall),
	)
	if err != nil {
		t.Fatalf("create all students install assignment: %v", err)
	}

	effective, err := stores.assignments.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 {
		t.Fatalf("effective packages = %+v, want one resolved item", effective)
	}
	if effective[0].Action != assignments.AssignmentActionRemove || effective[0].Package.Name != "SchoolApp" {
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "RowOrderApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	installPackage := createMunkiPackage(t, ctx, stores, title.ID, "RowOrderApp", "1.0")
	removePackage := createMunkiPackage(t, ctx, stores, title.ID, "RowOrderApp", "2.0")

	_, err = stores.assignments.Create(ctx, includeSpecificAssignment(
		title.ID,
		1,
		installLabel.ID,
		assignments.AssignmentActionInstall,
		installPackage.ID,
	))
	if err != nil {
		t.Fatalf("create install assignment: %v", err)
	}
	_, err = stores.assignments.Create(ctx, includeSpecificAssignment(
		title.ID,
		2,
		removeLabel.ID,
		assignments.AssignmentActionRemove,
		removePackage.ID,
	))
	if err != nil {
		t.Fatalf("create remove assignment: %v", err)
	}

	effective, err := stores.assignments.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 {
		t.Fatalf("effective packages = %+v, want one resolved item", effective)
	}
	if effective[0].Action != assignments.AssignmentActionInstall || effective[0].Package.Version != "1.0" {
		t.Fatalf("effective pkg = %+v, want first row install of RowOrderApp 1.0", effective[0])
	}
}

func TestCreatePackageRejectsUnsupportedArchitecture(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "Broken"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:             title.ID,
		Name:                   "Broken",
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

	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "MissingRelationApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	missingPackageID := int64(0)

	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID: title.ID,
		Name:       "MissingRelationApp",
		Version:    "1.0",
		Requires:   []packages.PackageReference{{PackageID: &missingPackageID}},
		Eligible:   true,
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("CreatePackage error = %v, want ErrNotFound", err)
	}
}

func TestCreatePackageBadInstalledSizeFallsThroughToInvalidInput(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "BadInstalledSizeApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	_, err = stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:    title.ID,
		Name:          "BadInstalledSizeApp",
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

	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "ExtraApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	dependency := createMunkiPackage(t, ctx, stores, title.ID, "DependencyApp", "2.0")
	pkg, err := stores.packages.Create(ctx, packages.PackageMutation{
		SoftwareID:         title.ID,
		Name:               "ExtraApp",
		Version:            "1.0",
		InstallerType:      packages.InstallerTypeNoPkg,
		InstallcheckScript: "#!/bin/zsh\nexit 0\n",
		Requires: []packages.PackageReference{
			{PackageID: &dependency.ID},
			{Name: "Python"},
		},
		Eligible: true,
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.InstallcheckScript == "" || pkg.InstallerType != packages.InstallerTypeNoPkg {
		t.Fatalf("pkg typed fields = %+v, want nopkg installcheck script", pkg)
	}
	if len(pkg.Requires) != 2 {
		t.Fatalf("requires = %+v, want dependency and literal reference", pkg.Requires)
	}
	if pkg.Requires[0].PackageID == nil || *pkg.Requires[0].PackageID != dependency.ID {
		t.Fatalf("first requires = %+v, want dependency package id", pkg.Requires[0])
	}
	if pkg.Requires[0].PackageName != "DependencyApp" || pkg.Requires[0].PackageVersion != "2.0" {
		t.Fatalf("first requires target = %+v, want dependency package details", pkg.Requires[0])
	}
	if pkg.Requires[1].Name != "Python" {
		t.Fatalf("second requires = %+v, want literal Python", pkg.Requires[1])
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

	pkg, err := stores.packages.Import(ctx, packages.PackageImportMutation{
		IconArtifactID: &iconArtifact.ID,
		Pkginfo: []byte(`{
			"name": "ImportedApp",
			"version": "1.2.3",
			"display_name": "Imported App",
			"description": "Managed by AutoPkg",
			"category": "Utilities",
			"developer": "Example",
			"installer_type": "nopkg",
			"unattended_install": true,
			"supported_architectures": ["arm64", "x86_64"],
			"requires": ["Python"],
			"icon_name": "ImportedApp.png",
			"icon_hash": "stale",
			"installs": [{"path": "/Applications/Imported App.app"}],
			"installer_item_location": "pkgs/ImportedApp.pkg"
		}`),
	})
	if err != nil {
		t.Fatalf("import package: %v", err)
	}
	if pkg.Name != "ImportedApp" || pkg.SoftwareName != "ImportedApp" ||
		pkg.InstallerType != packages.InstallerTypeNoPkg {
		t.Fatalf("pkg = %+v, want imported typed package", pkg)
	}
	if !pkg.UnattendedInstall || !sameStrings(pkg.SupportedArchitectures, []string{"arm64", "x86_64"}) {
		t.Fatalf("pkg typed fields = %+v", pkg)
	}
	if len(pkg.Requires) != 1 || pkg.Requires[0].Name != "Python" {
		t.Fatalf("requires = %+v, want literal Python", pkg.Requires)
	}
	if len(pkg.Installs) != 1 || pkg.Installs[0].Path != "/Applications/Imported App.app" {
		t.Fatalf("installs = %+v, want imported install item", pkg.Installs)
	}
	if pkg.IconName != "cccccccccccc/ImportedApp.png" || pkg.IconHash != strings.Repeat("c", 64) {
		t.Fatalf("pkg icon fields = name %q hash %q, want artifact-backed icon", pkg.IconName, pkg.IconHash)
	}

	updated, err := stores.packages.Import(ctx, packages.PackageImportMutation{
		Pkginfo: []byte(`{
			"name": "ImportedApp",
			"version": "1.2.3",
			"display_name": "Imported App",
			"developer": "Example Updated",
			"installer_type": "nopkg"
		}`),
	})
	if err != nil {
		t.Fatalf("import package update: %v", err)
	}
	if updated.ID != pkg.ID {
		t.Fatalf("updated package id = %d, want upserted id %d", updated.ID, pkg.ID)
	}
	if updated.Developer != "Example Updated" {
		t.Fatalf("updated developer = %q, want import update", updated.Developer)
	}
}

func TestUpdatePackagePreservesTypedImportFields(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	pkg, err := stores.packages.Import(ctx, packages.PackageImportMutation{
		Pkginfo: []byte(`{
			"name": "ImportedEditApp",
			"version": "1.2.3",
			"display_name": "Imported Edit App",
			"installer_type": "nopkg",
			"minimum_munki_version": "6.6",
			"blocking_applications": ["Safari"],
			"requires": ["Python"],
			"installs": [{"path": "/Applications/Imported Edit App.app"}]
		}`),
	})
	if err != nil {
		t.Fatalf("import package: %v", err)
	}

	updated, err := stores.packages.Update(ctx, pkg.ID, packages.PackageMutation{
		SoftwareID:             pkg.SoftwareID,
		Name:                   pkg.Name,
		Version:                "1.2.4",
		DisplayName:            "Imported Edit App Updated",
		InstallerType:          pkg.InstallerType,
		UnattendedInstall:      pkg.UnattendedInstall,
		UnattendedUninstall:    pkg.UnattendedUninstall,
		Uninstallable:          pkg.Uninstallable,
		UninstallMethod:        pkg.UninstallMethod,
		RestartAction:          pkg.RestartAction,
		MinimumOSVersion:       pkg.MinimumOSVersion,
		MaximumOSVersion:       pkg.MaximumOSVersion,
		SupportedArchitectures: pkg.SupportedArchitectures,
		OnDemand:               pkg.OnDemand,
		Precache:               pkg.Precache,
		Eligible:               pkg.Eligible,
	})
	if err != nil {
		t.Fatalf("update package: %v", err)
	}
	if updated.MinimumMunkiVersion != "6.6" {
		t.Fatalf("minimum munki version = %q, want preserved", updated.MinimumMunkiVersion)
	}
	if !sameStrings(updated.BlockingApplications, []string{"Safari"}) {
		t.Fatalf("blocking applications = %v, want preserved", updated.BlockingApplications)
	}
	if !samePackageReferenceNames(updated.Requires, []string{"Python"}) {
		t.Fatalf("requires = %v, want preserved", updated.Requires)
	}
	if len(updated.Installs) != 1 || updated.Installs[0].Path != "/Applications/Imported Edit App.app" {
		t.Fatalf("installs = %+v, want preserved imported install item", updated.Installs)
	}
}

func TestUpdatePackageRejectsSoftwareMove(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)

	first, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "FirstApp"})
	if err != nil {
		t.Fatalf("create first title: %v", err)
	}
	second, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "SecondApp"})
	if err != nil {
		t.Fatalf("create second title: %v", err)
	}
	pkg := createMunkiPackage(t, ctx, stores, first.ID, "FirstApp", "1.0")

	_, err = stores.packages.Update(ctx, pkg.ID, packages.PackageMutation{
		SoftwareID:    second.ID,
		Name:          pkg.Name,
		Version:       pkg.Version,
		InstallerType: pkg.InstallerType,
		Eligible:      pkg.Eligible,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("UpdatePackage error = %v, want invalid input", err)
	}
}

func TestCreateAssignmentTargetsAllHostsLabel(t *testing.T) {
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
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "AllDevicesApp"})
	if err != nil {
		t.Fatalf("create title: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, "AllDevicesApp", "1.0")
	_, err = stores.assignments.Create(ctx, includeAssignment(
		title.ID,
		1,
		allHostsLabelID(t, ctx, labelStore),
		assignments.AssignmentActionInstall,
	))
	if err != nil {
		t.Fatalf("create all-hosts assignment: %v", err)
	}
	effective, err := stores.assignments.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 || effective[0].Package.Name != "AllDevicesApp" {
		t.Fatalf("effective packages = %+v, want AllDevicesApp", effective)
	}
}

func TestUpdateAssignmentRejectsSoftwareMove(t *testing.T) {
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	first, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "FirstAssignedApp"})
	if err != nil {
		t.Fatalf("create first title: %v", err)
	}
	second, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "SecondAssignedApp"})
	if err != nil {
		t.Fatalf("create second title: %v", err)
	}
	pkg := createMunkiPackage(t, ctx, stores, first.ID, "FirstAssignedApp", "1.0")
	assignment, err := stores.assignments.Create(ctx, includeSpecificAssignment(
		first.ID,
		1,
		allHostsLabelID(t, ctx, labelStore),
		assignments.AssignmentActionInstall,
		pkg.ID,
	))
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	mutation := includeSpecificAssignment(
		second.ID,
		assignment.Priority,
		assignment.LabelID,
		assignment.Action,
		*assignment.PinnedPackageID,
	)
	_, err = stores.assignments.Update(ctx, assignment.ID, mutation)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("UpdateAssignment error = %v, want invalid input", err)
	}
}

func TestAssignmentExcludeLabelsRejectBuiltinAndIncludeOverlap(t *testing.T) {
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	title, err := stores.softwareTitles.Create(
		ctx,
		softwaretitles.SoftwareTitleMutation{Name: "ExcludeOverlapApp"},
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
	if _, err := stores.assignments.ReplaceExcludeLabelIDs(
		ctx,
		title.ID,
		[]int64{allHostsLabelID(t, ctx, labelStore)},
	); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("ReplaceExcludeLabelIDs builtin error = %v, want ErrInvalidInput", err)
	}
	if _, err := stores.assignments.Create(ctx, includeAssignment(
		title.ID,
		1,
		includeLabel.ID,
		assignments.AssignmentActionInstall,
	)); err != nil {
		t.Fatalf("create include assignment: %v", err)
	}
	if _, err := stores.assignments.ReplaceExcludeLabelIDs(ctx, title.ID, []int64{includeLabel.ID}); !errors.Is(
		err,
		dbutil.ErrInvalidInput,
	) {
		t.Fatalf("ReplaceExcludeLabelIDs overlap error = %v, want ErrInvalidInput", err)
	}
	if _, err := stores.assignments.ReplaceExcludeLabelIDs(ctx, title.ID, []int64{excludeLabel.ID}); err != nil {
		t.Fatalf("set exclude labels: %v", err)
	}
	if _, err := stores.assignments.Create(ctx, includeAssignment(
		title.ID,
		2,
		excludeLabel.ID,
		assignments.AssignmentActionInstall,
	)); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Create include for excluded label error = %v, want ErrInvalidInput", err)
	}
}

func TestDeleteSoftwareTitlesCleansPackagesAssignmentsAndIgnoresMissingBulkIDs(t *testing.T) {
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)
	labelID := allHostsLabelID(t, ctx, labelStore)

	first, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "DeletePinnedApp"})
	if err != nil {
		t.Fatalf("create first title: %v", err)
	}
	firstPkg := createMunkiPackage(t, ctx, stores, first.ID, "DeletePinnedApp", "1.0")
	if _, err := stores.assignments.Create(ctx, includeSpecificAssignment(
		first.ID,
		1,
		labelID,
		assignments.AssignmentActionInstall,
		firstPkg.ID,
	)); err != nil {
		t.Fatalf("create first assignment: %v", err)
	}

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

	second, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "BulkPinnedApp"})
	if err != nil {
		t.Fatalf("create second title: %v", err)
	}
	secondPkg := createMunkiPackage(t, ctx, stores, second.ID, "BulkPinnedApp", "1.0")
	if _, err := stores.assignments.Create(ctx, includeSpecificAssignment(
		second.ID,
		1,
		labelID,
		assignments.AssignmentActionInstall,
		secondPkg.ID,
	)); err != nil {
		t.Fatalf("create second assignment: %v", err)
	}
	third, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "BulkPlainApp"})
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

func TestAssignmentMissingLabelFallsThroughToNotFound(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newMunkiStores(db)
	title, err := stores.softwareTitles.Create(
		ctx,
		softwaretitles.SoftwareTitleMutation{Name: "MissingLabelAssignment"},
	)
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	_, err = stores.assignments.Create(ctx, includeAssignment(title.ID, 1, 0, assignments.AssignmentActionInstall))
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("CreateAssignment error = %v, want ErrNotFound", err)
	}
}

func TestAssignmentBadPriorityFallsThroughToInvalidInput(t *testing.T) {
	db, ctx := dbtest.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)
	title, err := stores.softwareTitles.Create(ctx, softwaretitles.SoftwareTitleMutation{Name: "BadPriorityAssignment"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Bad Priority Label",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	_, err = stores.assignments.Create(
		ctx,
		includeAssignment(title.ID, 0, label.ID, assignments.AssignmentActionInstall),
	)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreateAssignment error = %v, want ErrInvalidInput", err)
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

func includeAssignment(
	softwareID int64,
	priority int32,
	labelID int64,
	action assignments.AssignmentAction,
) assignments.AssignmentMutation {
	selection := assignments.PackageSelectionLatestEligible
	return assignments.AssignmentMutation{
		SoftwareID:       softwareID,
		Priority:         priority,
		LabelID:          labelID,
		Action:           action,
		PackageSelection: selection,
	}
}

func includeSpecificAssignment(
	softwareID int64,
	priority int32,
	labelID int64,
	action assignments.AssignmentAction,
	pinnedPackageID int64,
) assignments.AssignmentMutation {
	selection := assignments.PackageSelectionSpecific
	return assignments.AssignmentMutation{
		SoftwareID:       softwareID,
		Priority:         priority,
		LabelID:          labelID,
		Action:           action,
		PackageSelection: selection,
		PinnedPackageID:  &pinnedPackageID,
	}
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

func samePackageReferenceNames(a []packages.PackageReference, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i] {
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
		Name:          name,
		Version:       version,
		InstallerType: packages.InstallerTypeNoPkg,
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
	assignmentRows, assignmentCount, err := stores.assignments.List(
		ctx,
		assignments.AssignmentListParams{SoftwareID: softwareID},
	)
	if err != nil {
		t.Fatalf("list assignments after delete: %v", err)
	}
	if assignmentCount != 0 || len(assignmentRows) != 0 {
		t.Fatalf("assignments after delete = %+v count = %d, want none", assignmentRows, assignmentCount)
	}
	excludeLabelIDs, err := stores.assignments.ExcludeLabelIDs(ctx, softwareID)
	if err != nil {
		t.Fatalf("list exclude labels after delete: %v", err)
	}
	if len(excludeLabelIDs) != 0 {
		t.Fatalf("exclude labels after delete = %v, want none", excludeLabelIDs)
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
