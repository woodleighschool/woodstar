//go:build postgres

package munki_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/woodleighschool/goodies/bloby"
	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestMunkiSoftwareExclusionOverridesAllHostsInclude(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	stores := newMunkiStores(t, db)

	excludedHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-desired-excluded-uuid", Serial: "C02MUNKIOUT"},
		OrbitNodeKey: "munki-desired-excluded-orbit",
	}, heartbeats.Contact{})

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
		packages.PackageCreateMutation{SoftwareID: title.ID,
			Version:       "148.0.0.1",
			InstallerType: packages.InstallerTypeNoPkg,
			OnDemand:      true},
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

func TestEffectivePackagesForHostKeepsLatestCandidates(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	stores := newMunkiStores(t, db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-latest-host-uuid", Serial: "C02MUNKILATEST"},
		OrbitNodeKey: "munki-latest-orbit",
	}, heartbeats.Contact{})

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

func TestRepositoryServiceScopesCatalogAndFilesByHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	firstHostID, firstLabelID := createRepositoryScopeHost(t, ctx, hostStore, labelStore, "first", "1")
	secondHostID, secondLabelID := createRepositoryScopeHost(t, ctx, hostStore, labelStore, "second", "2")
	firstNames := createFirstRepositoryScope(t, ctx, stores, firstLabelID)
	second := createSecondRepositoryScope(t, ctx, stores, secondLabelID)
	service := munki.NewRepositoryService(munki.Dependencies{
		Software: stores.software,
		Packages: stores.packages,
		Objects:  stores.objects,
	})

	t.Run("first host excludes second host resources", func(t *testing.T) {
		assertFirstRepositoryScope(t, ctx, service, firstHostID, firstNames, second)
	})
	t.Run("second host sees its own resources", func(t *testing.T) {
		assertSecondRepositoryScope(t, ctx, service, secondHostID, second)
	})
}

type repositoryScopePackage struct {
	pkg       *packages.Package
	installer *bloby.Object
	icon      *bloby.Object
}

func createRepositoryScopeHost(
	t *testing.T,
	ctx context.Context,
	hostStore *hosts.Store,
	labelStore *labels.Store,
	name, serialSuffix string,
) (int64, int64) {
	t.Helper()
	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:   "repository-" + name + "-host",
			Serial: "C02REPOSITORY" + serialSuffix,
		},
		OrbitNodeKey: "repository-" + name + "-orbit",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll %s host: %v", name, err)
	}
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Repository " + name + " Host",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{host.ID},
	})
	if err != nil {
		t.Fatalf("create %s label: %v", name, err)
	}
	return host.ID, label.ID
}

func createFirstRepositoryScope(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	labelID int64,
) []string {
	t.Helper()
	dependencySoftware, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "RepositoryDependency"})
	if err != nil {
		t.Fatalf("create dependency software: %v", err)
	}
	dependency := createMunkiPackage(t, ctx, stores, dependencySoftware.ID, dependencySoftware.Name, "1.0")
	firstIcon := createMunkiIconObject(t, ctx, stores, "RepositoryFirst.png", "e")
	firstLatestSoftware, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:         "RepositoryFirstLatest",
		IconObjectID: &firstIcon.ID,
	})
	if err != nil {
		t.Fatalf("create first latest software: %v", err)
	}
	firstLatest, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID:    firstLatestSoftware.ID,
		Version:       "1.0",
		InstallerType: packages.InstallerTypeNoPkg,
		Requires:      []packages.PackageReferenceMutation{{SoftwareID: dependencySoftware.ID}},
	},
	)
	if err != nil {
		t.Fatalf("create first latest package: %v", err)
	}
	pinnedSoftware, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "RepositoryPinned"})
	if err != nil {
		t.Fatalf("create pinned software: %v", err)
	}
	pinned := createMunkiPackage(t, ctx, stores, pinnedSoftware.ID, pinnedSoftware.Name, "1.0")
	replaceTargets(t, ctx, stores, firstLatestSoftware, []munkisoftware.Include{
		includeTarget(labelID, munkisoftware.ActionManagedInstalls),
	})
	replaceTargets(t, ctx, stores, pinnedSoftware, []munkisoftware.Include{
		includeSpecificTarget(labelID, munkisoftware.ActionManagedInstalls, pinned.ID),
	})
	return []string{firstLatest.Software.Name, pinned.Software.Name, dependency.Software.Name}
}

func createSecondRepositoryScope(
	t *testing.T,
	ctx context.Context,
	stores munkiStores,
	labelID int64,
) repositoryScopePackage {
	t.Helper()
	icon := createMunkiIconObject(t, ctx, stores, "RepositorySecond.png", "f")
	software, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:         "RepositorySecond",
		IconObjectID: &icon.ID,
	})
	if err != nil {
		t.Fatalf("create second software: %v", err)
	}
	installer := createMunkiPackageObject(t, ctx, stores, "RepositorySecond.pkg", "a")
	pkg, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID:        software.ID,
		Version:           "2.0",
		InstallerObjectID: &installer.ID,
	},
	)
	if err != nil {
		t.Fatalf("create second package: %v", err)
	}
	replaceTargets(t, ctx, stores, software, []munkisoftware.Include{
		includeTarget(labelID, munkisoftware.ActionManagedInstalls),
	})
	return repositoryScopePackage{pkg: pkg, installer: installer, icon: icon}
}

func assertFirstRepositoryScope(
	t *testing.T,
	ctx context.Context,
	service *munki.RepositoryService,
	hostID int64,
	wantNames []string,
	other repositoryScopePackage,
) {
	t.Helper()
	gotNames := repositoryCatalogNames(t, ctx, service, hostID)
	for _, name := range wantNames {
		if !slices.Contains(gotNames, name) {
			t.Fatalf("first catalog names = %v, want %q", gotNames, name)
		}
	}
	if slices.Contains(gotNames, other.pkg.Software.Name) {
		t.Fatalf("first catalog names = %v, must exclude %q", gotNames, other.pkg.Software.Name)
	}
	if _, err := service.ResolvePackageFile(ctx, hostID, packages.InstallerItemLocation(*other.pkg, *other.installer)); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("first host second package error = %v, want ErrNotFound", err)
	}
	if _, err := service.ResolveIconFile(ctx, hostID, packages.IconName(*other.icon)); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("first host second icon error = %v, want ErrNotFound", err)
	}
}

func assertSecondRepositoryScope(
	t *testing.T,
	ctx context.Context,
	service *munki.RepositoryService,
	hostID int64,
	want repositoryScopePackage,
) {
	t.Helper()
	names := repositoryCatalogNames(t, ctx, service, hostID)
	if len(names) != 1 || names[0] != want.pkg.Software.Name {
		t.Fatalf("second catalog = %v, want %q", names, want.pkg.Software.Name)
	}
	if _, err := service.ResolvePackageFile(ctx, hostID, packages.InstallerItemLocation(*want.pkg, *want.installer)); err != nil {
		t.Fatalf("second host package: %v", err)
	}
	if _, err := service.ResolveIconFile(ctx, hostID, packages.IconName(*want.icon)); err != nil {
		t.Fatalf("second host icon: %v", err)
	}
}

func repositoryCatalogNames(
	t *testing.T,
	ctx context.Context,
	service *munki.RepositoryService,
	hostID int64,
) []string {
	t.Helper()
	catalog, err := service.Catalog(ctx, hostID, "woodstar")
	if err != nil {
		t.Fatalf("host %d catalog: %v", hostID, err)
	}
	var items []struct {
		Name string `plist:"name"`
	}
	if _, err := plist.Unmarshal(catalog, &items); err != nil {
		t.Fatalf("decode host %d catalog: %v", hostID, err)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func TestEffectivePackagesForHostUsesPriorityForSchoolTargets(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	stores := newMunkiStores(t, db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-sac-student-uuid", Serial: "C02MUNKISAC"},
		OrbitNodeKey: "munki-sac-student-orbit",
	}, heartbeats.Contact{})

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
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	stores := newMunkiStores(t, db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-row-order-uuid", Serial: "C02MUNKIRO"},
		OrbitNodeKey: "munki-row-order-orbit",
	}, heartbeats.Contact{})

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

func TestHostMunkiStateKeepsDesiredSoftwareSeparateFromExactObservations(t *testing.T) { //nolint:cyclop,funlen,gocognit // One desired/observed database lifecycle.
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	stores := newMunkiStores(t, db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-host-observation-uuid", Serial: "C02MUNKI"},
		OrbitNodeKey: "munki-host-observation-orbit",
	}, heartbeats.Contact{})

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
	infoRows := []map[string]string{{
		"version":          "7.1.2.5700",
		"manifest_name":    "site_default",
		"errors":           "first error",
		"warnings":         "first warning",
		"problem_installs": "Broken App",
		"start_time":       runStartedAt.Format("2006-01-02 15:04:05 -0700"),
		"end_time":         runEndedAt.Format("2006-01-02 15:04:05 -0700"),
	}}
	collector := munki.NewDetailIngestor(stores.hoststate)
	if err := collector.IngestCollection(ctx, host.ID, munki.Collection{
		Info: munki.QueryResult{Present: true, Successful: true, Rows: infoRows},
		Installs: munki.QueryResult{Present: true, Successful: true, Rows: []map[string]string{
			{
				"name":               "NotGoogleChrome",
				"display_name":       "GoogleChrome",
				"version_to_install": "148.0",
			},
			{
				"name":               "VisualStudioCode",
				"display_name":       "Visual Studio Code",
				"installed":          "false",
				"installed_version":  "",
				"version_to_install": "1.130.0",
			},
		}},
	}); err != nil {
		t.Fatalf("ingest Munki collection: %v", err)
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
	if !slices.Equal(detail.ProblemInstalls, []string{"Broken App"}) {
		t.Fatalf("problem installs = %v, want [Broken App]", detail.ProblemInstalls)
	}
	if detail.RunAt == nil || !detail.RunAt.Equal(runEndedAt) {
		t.Fatalf("detail run time = %v, want completion time %v", detail.RunAt, runEndedAt)
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

	lastRunAt := *detail.RunAt
	if err := collector.IngestCollection(ctx, host.ID, munki.Collection{
		Info: munki.QueryResult{Present: true, Successful: true, Rows: infoRows},
	}); err != nil {
		t.Fatalf("ingest incomplete Munki collection: %v", err)
	}
	detail, err = stores.hoststate.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load failed Munki collection: %v", err)
	}
	if detail == nil || detail.RunAt == nil || !detail.RunAt.Equal(lastRunAt) ||
		detail.Version != "7.1.2.5700" {
		t.Fatalf("failed collection state = %+v, want retained successful report", detail)
	}
	if !slices.Equal(detail.ProblemInstalls, []string{"Broken App"}) {
		t.Fatalf("problem installs after failed collection = %v, want retained report", detail.ProblemInstalls)
	}
	var observedCount int
	if err := stores.db.QueryRow(
		ctx,
		`SELECT count(*) FROM munki_host_items WHERE host_id = $1`,
		host.ID,
	).Scan(&observedCount); err != nil {
		t.Fatalf("count observations after failed collection: %v", err)
	}
	if observedCount != 2 {
		t.Fatalf("observations after failed collection = %d, want 2", observedCount)
	}

	nextVSCodePackage := createMunkiPackage(t, ctx, stores, vscode.ID, vscode.Name, "1.131.0")
	replaceTargets(t, ctx, stores, vscode, []munkisoftware.Include{
		includeSpecificTarget(
			allHostsID,
			munkisoftware.ActionManagedUpdates,
			nextVSCodePackage.ID,
		),
	})
	if err := collector.IngestCollection(ctx, host.ID, munki.Collection{
		Info: munki.QueryResult{Present: true, Successful: true, Rows: infoRows},
		Installs: munki.QueryResult{Present: true, Successful: true, Rows: []map[string]string{{
			"name":              "VisualStudioCode",
			"display_name":      "Visual Studio Code",
			"installed":         "true",
			"installed_version": "1.130.0",
		}}},
	}); err != nil {
		t.Fatalf("ingest updated Munki collection: %v", err)
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

	if err := collector.IngestCollection(ctx, host.ID, munki.Collection{}); err != nil {
		t.Fatalf("ingest no-report Munki collection: %v", err)
	}
	if detail, err := stores.hoststate.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load no-report Munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("no-report Munki detail = %+v, want none", detail)
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
