package munki_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
)

func TestDesiredStateCreateListAndResolveForHost(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := munki.NewStore(db)

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
		HostIDs:             []int64{includedHost.ID, excludedHost.ID},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	title, err := store.CreateSoftwareTitle(ctx, munki.SoftwareTitleMutation{
		Name:        "GoogleChrome",
		DisplayName: "Google Chrome",
		Category:    "Browsers",
		Developer:   "Google",
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	pkg, err := store.CreatePackage(ctx, munki.PackageMutation{
		SoftwareID:    title.ID,
		Name:          "GoogleChrome",
		Version:       "148.0.0.1",
		InstallerType: munki.InstallerTypeNoPkg,
		Eligible:      true,
	})
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	deployment, err := store.CreateDeployment(ctx, munki.DeploymentMutation{
		PackageID:       pkg.ID,
		Intent:          munki.IntentEnsureInstalled,
		IncludeLabelIDs: []int64{label.ID},
		ExcludeHostIDs:  []int64{excludedHost.ID},
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	titles, titleCount, err := store.ListSoftwareTitles(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list software titles: %v", err)
	}
	if titleCount != 1 || len(titles) != 1 || titles[0].Name != "GoogleChrome" {
		t.Fatalf("titles = %+v count = %d, want GoogleChrome", titles, titleCount)
	}
	packages, pkgCount, err := store.ListPackages(ctx, munki.PackageListParams{})
	if err != nil {
		t.Fatalf("list packages: %v", err)
	}
	if pkgCount != 1 || len(packages) != 1 || packages[0].Version != "148.0.0.1" {
		t.Fatalf("packages = %+v count = %d, want version 148.0.0.1", packages, pkgCount)
	}
	deployments, deploymentCount, err := store.ListDeployments(ctx, munki.DeploymentListParams{})
	if err != nil {
		t.Fatalf("list deployments: %v", err)
	}
	if deploymentCount != 1 || len(deployments) != 1 || deployments[0].ID != deployment.ID {
		t.Fatalf("deployments = %+v count = %d, want created deployment", deployments, deploymentCount)
	}
	if !sameInt64s(deployments[0].IncludeLabelIDs, []int64{label.ID}) {
		t.Fatalf("include label ids = %v, want %v", deployments[0].IncludeLabelIDs, []int64{label.ID})
	}
	if !sameInt64s(deployments[0].ExcludeHostIDs, []int64{excludedHost.ID}) {
		t.Fatalf("exclude host ids = %v, want %v", deployments[0].ExcludeHostIDs, []int64{excludedHost.ID})
	}

	included, err := store.EffectivePackagesForHost(ctx, includedHost.ID)
	if err != nil {
		t.Fatalf("resolve included host: %v", err)
	}
	if len(included) != 1 || included[0].Package.Name != "GoogleChrome" ||
		included[0].Intent != munki.IntentEnsureInstalled {
		t.Fatalf("included effective packages = %+v, want GoogleChrome install", included)
	}
	excluded, err := store.EffectivePackagesForHost(ctx, excludedHost.ID)
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
	store := munki.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-artifact-host-uuid", Serial: "C02MUNKIART"},
		OrbitNodeKey: "munki-artifact-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	title, err := store.CreateSoftwareTitle(ctx, munki.SoftwareTitleMutation{Name: "ArtifactApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	artifact, err := store.CreateArtifact(ctx, munki.ArtifactMutation{
		Kind:        munki.ArtifactKindPackage,
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
	loadedArtifact, err := store.GetArtifact(ctx, artifact.ID)
	if err != nil {
		t.Fatalf("get artifact: %v", err)
	}
	if loadedArtifact.StorageKey != "pkgs/ArtifactApp.pkg" {
		t.Fatalf("storage key = %q", loadedArtifact.StorageKey)
	}
	loadedByLocation, err := store.GetArtifactByLocation(ctx, munki.ArtifactKindPackage, "apps/ArtifactApp.pkg")
	if err != nil {
		t.Fatalf("get artifact by location: %v", err)
	}
	if loadedByLocation.ID != artifact.ID {
		t.Fatalf("artifact by location id = %d, want %d", loadedByLocation.ID, artifact.ID)
	}
	artifacts, artifactCount, err := store.ListArtifacts(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if artifactCount != 1 || len(artifacts) != 1 || artifacts[0].ID != artifact.ID {
		t.Fatalf("artifacts = %+v count = %d, want created artifact", artifacts, artifactCount)
	}

	pkg, err := store.CreatePackage(ctx, munki.PackageMutation{
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
	if _, err := store.CreateDeployment(ctx, munki.DeploymentMutation{
		PackageID: pkg.ID,
		Intent:    munki.IntentEnsureInstalled,
		AllHosts:  true,
	}); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	effective, err := store.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 || effective[0].Package.InstallerArtifactLocation != "apps/ArtifactApp.pkg" {
		t.Fatalf("effective packages = %+v, want artifact location", effective)
	}
}

func TestCreatePackageRejectsIconArtifactAsInstaller(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := munki.NewStore(db)

	title, err := store.CreateSoftwareTitle(ctx, munki.SoftwareTitleMutation{Name: "IconApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	artifact, err := store.CreateArtifact(ctx, munki.ArtifactMutation{
		Kind:       munki.ArtifactKindIcon,
		Location:   "IconApp.png",
		SizeBytes:  256,
		SHA256:     strings.Repeat("b", 64),
		StorageKey: "icons/IconApp.png",
	})
	if err != nil {
		t.Fatalf("create icon artifact: %v", err)
	}

	_, err = store.CreatePackage(ctx, munki.PackageMutation{
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

func TestEffectivePackagesForHostResolvesOverlappingDeployments(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := munki.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-overlap-uuid", Serial: "C02MUNKIOL"},
		OrbitNodeKey: "munki-overlap-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Munki Overlap Test",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{host.ID},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	title, err := store.CreateSoftwareTitle(ctx, munki.SoftwareTitleMutation{Name: "OverlapApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	optionalPackage := createMunkiPackage(t, ctx, store, title.ID, "OverlapApp", "1.0")
	installPackage := createMunkiPackage(t, ctx, store, title.ID, "OverlapApp", "2.0")
	absentPackage := createMunkiPackage(t, ctx, store, title.ID, "OverlapApp", "3.0")

	optionalDeployment, err := store.CreateDeployment(ctx, munki.DeploymentMutation{
		PackageID: optionalPackage.ID,
		Intent:    munki.IntentOptional,
		AllHosts:  true,
	})
	if err != nil {
		t.Fatalf("create all-host optional deployment: %v", err)
	}
	installDeployment, err := store.CreateDeployment(ctx, munki.DeploymentMutation{
		PackageID:       installPackage.ID,
		Intent:          munki.IntentEnsureInstalled,
		IncludeLabelIDs: []int64{label.ID},
	})
	if err != nil {
		t.Fatalf("create label install deployment: %v", err)
	}
	absentDeployment, err := store.CreateDeployment(ctx, munki.DeploymentMutation{
		PackageID:      absentPackage.ID,
		Intent:         munki.IntentEnsureAbsent,
		IncludeHostIDs: []int64{host.ID},
	})
	if err != nil {
		t.Fatalf("create host removal deployment: %v", err)
	}
	if err := store.ReorderDeployments(ctx, title.ID, []int64{
		absentDeployment.ID,
		installDeployment.ID,
		optionalDeployment.ID,
	}); err != nil {
		t.Fatalf("reorder deployments: %v", err)
	}

	effective, err := store.EffectivePackagesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective packages: %v", err)
	}
	if len(effective) != 1 {
		t.Fatalf("effective packages = %+v, want one resolved item", effective)
	}
	if effective[0].Intent != munki.IntentEnsureAbsent || effective[0].Package.Version != "3.0" {
		t.Fatalf("effective pkg = %+v, want removal of OverlapApp 3.0", effective[0])
	}
}

func TestCreatePackageRejectsInvalidPkginfoFields(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := munki.NewStore(db)

	title, err := store.CreateSoftwareTitle(ctx, munki.SoftwareTitleMutation{Name: "Broken"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}

	_, err = store.CreatePackage(ctx, munki.PackageMutation{
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

func TestPackageExtraPkginfoCannotOverrideTypedFields(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := munki.NewStore(db)

	title, err := store.CreateSoftwareTitle(ctx, munki.SoftwareTitleMutation{Name: "ExtraApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	pkg, err := store.CreatePackage(ctx, munki.PackageMutation{
		SoftwareID:   title.ID,
		Name:         "ExtraApp",
		Version:      "1.0",
		ExtraPkginfo: []byte(`{"installer_type":"nopkg","unattended_install":true,"custom_key":"kept"}`),
		Eligible:     true,
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(pkg.Pkginfo, &rendered); err != nil {
		t.Fatalf("decode pkginfo: %v", err)
	}
	if _, ok := rendered["installer_type"]; ok {
		t.Fatalf("pkginfo = %s, want typed default package to omit installer_type", pkg.Pkginfo)
	}
	if _, ok := rendered["unattended_install"]; ok {
		t.Fatalf("pkginfo = %s, want typed false to omit unattended_install", pkg.Pkginfo)
	}
	if rendered["custom_key"] != "kept" {
		t.Fatalf("pkginfo custom_key = %v, want preserved extra key", rendered["custom_key"])
	}
}

func TestImportPackageUpsertsTypedPkginfo(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := munki.NewStore(db)

	iconArtifact, err := store.CreateArtifact(ctx, munki.ArtifactMutation{
		Kind:       munki.ArtifactKindIcon,
		Location:   "cccccccccccc/ImportedApp.png",
		SizeBytes:  256,
		SHA256:     strings.Repeat("c", 64),
		StorageKey: "icons/cccccccccccc/ImportedApp.png",
	})
	if err != nil {
		t.Fatalf("create icon artifact: %v", err)
	}

	pkg, err := store.ImportPackage(ctx, munki.PackageImportMutation{
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
	if pkg.Name != "ImportedApp" || pkg.SoftwareName != "ImportedApp" || pkg.InstallerType != munki.InstallerTypeNoPkg {
		t.Fatalf("pkg = %+v, want imported typed package", pkg)
	}
	if !pkg.UnattendedInstall || !sameStrings(pkg.SupportedArchitectures, []string{"arm64", "x86_64"}) {
		t.Fatalf("pkg typed fields = %+v", pkg)
	}
	if !strings.Contains(string(pkg.ExtraPkginfo), `"installs"`) {
		t.Fatalf("extra pkginfo = %s, want installs preserved", pkg.ExtraPkginfo)
	}
	if pkg.IconName != "cccccccccccc/ImportedApp.png" || pkg.IconHash != strings.Repeat("c", 64) {
		t.Fatalf("pkg icon fields = name %q hash %q, want artifact-backed icon", pkg.IconName, pkg.IconHash)
	}
	if !strings.Contains(string(pkg.Pkginfo), `"icon_name":"cccccccccccc/ImportedApp.png"`) {
		t.Fatalf("pkginfo = %s, want artifact-backed icon_name", pkg.Pkginfo)
	}
	if strings.Contains(string(pkg.ExtraPkginfo), "installer_item_location") {
		t.Fatalf("extra pkginfo = %s, want rendered URL fields owned by Woodstar", pkg.ExtraPkginfo)
	}

	updated, err := store.ImportPackage(ctx, munki.PackageImportMutation{
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

func TestCreateDeploymentRejectsEmptyScope(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := munki.NewStore(db)

	_, err := store.CreateDeployment(ctx, munki.DeploymentMutation{
		PackageID: 1,
		Intent:    munki.IntentEnsureInstalled,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreateDeployment error = %v, want invalid input", err)
	}
}

func TestHostStatusUpsertAndDetail(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := munki.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-host-observation-uuid", Serial: "C02MUNKI"},
		OrbitNodeKey: "munki-host-observation-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	if detail, err := store.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load absent munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("absent munki detail = %+v, want nil", detail)
	}

	success := true
	if err := store.UpsertHostStatus(ctx, munki.HostStatusObservation{
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
	if err := store.ReplaceHostItems(ctx, host.ID, []munki.HostItem{
		{Name: "GoogleChrome", Installed: true, InstalledVersion: "148.0", RunEndedAt: "2026-05-31 19:24:14 +1000"},
		{Name: "Optional App", Installed: false},
	}); err != nil {
		t.Fatalf("replace munki host items: %v", err)
	}

	detail, err := store.LoadHostState(ctx, host.ID)
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

	if err := store.ReplaceHostItems(
		ctx,
		host.ID,
		[]munki.HostItem{{Name: "Replacement", Installed: true}},
	); err != nil {
		t.Fatalf("replace munki host items again: %v", err)
	}
	detail, err = store.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load munki detail after replace: %v", err)
	}
	if len(detail.Items) != 1 || detail.Items[0].Name != "Replacement" {
		t.Fatalf("items after replace = %+v", detail.Items)
	}

	if err := store.ClearHostStatus(ctx, host.ID); err != nil {
		t.Fatalf("clear munki host status: %v", err)
	}
	if detail, err := store.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load cleared munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("cleared munki detail = %+v, want nil", detail)
	}
}

func sameInt64s(a, b []int64) bool {
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
	store *munki.Store,
	softwareID int64,
	name string,
	version string,
) munki.Package {
	t.Helper()
	pkg, err := store.CreatePackage(ctx, munki.PackageMutation{
		SoftwareID:    softwareID,
		Name:          name,
		Version:       version,
		InstallerType: munki.InstallerTypeNoPkg,
		Eligible:      true,
	})
	if err != nil {
		t.Fatalf("create pkg %s %s: %v", name, version, err)
	}
	return *pkg
}
