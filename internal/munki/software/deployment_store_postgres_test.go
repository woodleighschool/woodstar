//go:build postgres

package software

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestDeploymentProjectionKeepsInventoryAndMunkiIndependent(t *testing.T) {
	fixture := newDeploymentStoreFixture(t)

	summary, err := fixture.store.GetDeployment(fixture.ctx, fixture.softwareID)
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	want := DeploymentSummary{
		AssignedCount:  6,
		ObservedCount:  5,
		InstalledCount: 3,
		Packages: []PackageDeployment{
			{Version: "1.2", InstalledCount: 1},
			{Version: "1.3", AssignedCount: 6, ObservedCount: 5, InstalledCount: 1},
		},
	}
	if !reflect.DeepEqual(summary, want) {
		t.Fatalf("deployment summary = %+v, want %+v", summary, want)
	}

	hosts, hostCount, err := fixture.store.ListDeploymentHosts(
		fixture.ctx,
		fixture.softwareID,
		DeploymentHostListParams{ListParams: dbutil.ListParams{PageSize: 1000}},
	)
	if err != nil {
		t.Fatalf("ListDeploymentHosts: %v", err)
	}
	if hostCount != 6 || len(hosts) != 6 {
		t.Fatalf("assigned hosts = %d/%d, want 6", len(hosts), hostCount)
	}
	byName := make(map[string]DeploymentHost, len(hosts))
	for _, host := range hosts {
		byName[host.DisplayName] = host
	}

	installedOld := byName["Installed Old"]
	if installedOld.Status != StatusInstalled || installedOld.InstalledVersion != "1.2" ||
		installedOld.MunkiResult != MunkiResultInstallIndicated || installedOld.TargetVersion != "1.3" ||
		installedOld.LastCollectedAt == nil {
		t.Fatalf("installed old host = %+v, want observed 1.2 and Munki target 1.3", installedOld)
	}
	munkiOnly := byName["Munki Only"]
	if munkiOnly.Status != StatusNotInstalled || munkiOnly.InstalledVersion != "" ||
		munkiOnly.MunkiResult != MunkiResultNoInstallNeeded || munkiOnly.TargetVersion != "1.3" {
		t.Fatalf("Munki-only host = %+v, want independently not installed", munkiOnly)
	}
	inventoryOnly := byName["Inventory Only"]
	if inventoryOnly.Status != StatusInstalled || inventoryOnly.InstalledVersion != "1.3" ||
		inventoryOnly.MunkiResult != MunkiResultNotReported || inventoryOnly.TargetVersion != "1.3" {
		t.Fatalf("inventory-only host = %+v, want installed and Munki not reported", inventoryOnly)
	}
	if empty := byName["Empty Snapshot"]; empty.Status != StatusNotInstalled ||
		empty.MunkiResult != MunkiResultUnresolved {
		t.Fatalf("empty snapshot host = %+v, want not installed and unresolved", empty)
	}
	if noSnapshot := byName["No Snapshot"]; noSnapshot.Status != StatusUnknown ||
		noSnapshot.LastCollectedAt != nil {
		t.Fatalf("no-snapshot host = %+v, want unknown", noSnapshot)
	}
	if conflicting := byName["Conflicting Versions"]; conflicting.Status != StatusInstalled ||
		conflicting.InstalledVersion != "" {
		t.Fatalf("conflicting-version host = %+v, want installed without a version", conflicting)
	}

	assertDeploymentFilter(t, fixture, DeploymentHostListParams{
		ListParams: dbutil.ListParams{PageSize: 1000},
		Status:     new(StatusInstalled),
	}, "Conflicting Versions", "Installed Old", "Inventory Only")
	assertDeploymentFilter(t, fixture, DeploymentHostListParams{
		ListParams:  dbutil.ListParams{PageSize: 1000},
		MunkiResult: new(MunkiResultInstallIndicated),
	}, "Installed Old")
	assertDeploymentFilter(t, fixture, DeploymentHostListParams{
		ListParams: dbutil.ListParams{PageSize: 1000, Q: "serial-Inventory Only"},
	}, "Inventory Only")

	rows, count, err := fixture.store.ListWithDeployment(fixture.ctx, dbutil.ListParams{PageSize: 1})
	if err != nil {
		t.Fatalf("ListWithDeployment: %v", err)
	}
	if count != 1 || len(rows) != 1 || !reflect.DeepEqual(rows[0].Deployment, want) {
		t.Fatalf("software page = %+v count %d, want deployment summary %+v", rows, count, want)
	}
}

func TestDeploymentExpectedPathSelectsRawBuildVersion(t *testing.T) {
	fixture := newDeploymentStoreFixture(t)
	if _, err := fixture.db.Pool().Exec(fixture.ctx, `
UPDATE munki_software
SET installation_detector_version_source = 'bundle_version'
WHERE id = $1`, fixture.softwareID); err != nil {
		t.Fatalf("switch detector version source: %v", err)
	}

	hostID := fixture.hostIDs["Installed Old"]
	if _, err := fixture.db.Pool().Exec(fixture.ctx, `
UPDATE host_software_installed_paths
SET bundle_version = CASE installed_path
	WHEN '/Applications/Alpha.app' THEN '120'
	ELSE '999'
END
WHERE host_id = $1`, hostID); err != nil {
		t.Fatalf("set raw build versions: %v", err)
	}

	hosts, _, err := fixture.store.ListDeploymentHosts(
		fixture.ctx,
		fixture.softwareID,
		DeploymentHostListParams{
			ListParams: dbutil.ListParams{PageSize: 1000, Q: "Installed Old"},
		},
	)
	if err != nil {
		t.Fatalf("ListDeploymentHosts: %v", err)
	}
	if len(hosts) != 1 || hosts[0].InstalledVersion != "120" {
		t.Fatalf("expected-path host = %+v, want raw build version 120", hosts)
	}
}

func TestDeploymentWithoutDetectorIsUnknownButKeepsMunkiResult(t *testing.T) {
	fixture := newDeploymentStoreFixture(t)
	if _, err := fixture.db.Pool().Exec(fixture.ctx, `
UPDATE munki_software
SET installation_detector_bundle_identifier = NULL,
	installation_detector_expected_path = NULL,
	installation_detector_version_source = NULL,
	installation_detector_automatic = FALSE
WHERE id = $1`, fixture.softwareID); err != nil {
		t.Fatalf("clear detector: %v", err)
	}

	hosts, _, err := fixture.store.ListDeploymentHosts(
		fixture.ctx,
		fixture.softwareID,
		DeploymentHostListParams{
			ListParams: dbutil.ListParams{PageSize: 1000, Q: "Installed Old"},
		},
	)
	if err != nil {
		t.Fatalf("ListDeploymentHosts: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Status != StatusUnknown ||
		hosts[0].MunkiResult != MunkiResultInstallIndicated || hosts[0].TargetVersion != "1.3" {
		t.Fatalf("detectorless host = %+v, want unknown with Munki result", hosts)
	}
}

func TestDeploymentObservedInstallationEligibility(t *testing.T) {
	t.Run("specific nopkg is unknown in software and host detail", func(t *testing.T) {
		fixture := newDeploymentStoreFixture(t)
		fixture.makeOnlyPackageNopkg(t)

		hosts := fixture.listInstalledOldHost(t)
		if len(hosts) != 1 || hosts[0].Status != StatusUnknown ||
			hosts[0].MunkiResult != MunkiResultInstallIndicated {
			t.Fatalf("specific nopkg host = %+v, want unknown with Munki result", hosts)
		}
		hostSoftware, count, err := fixture.store.ListForHost(
			fixture.ctx,
			fixture.hostIDs["Installed Old"],
			HostManifestSoftwareListParams{},
		)
		if err != nil {
			t.Fatalf("ListForHost: %v", err)
		}
		if count != 1 || len(hostSoftware) != 1 || hostSoftware[0].Status != StatusUnknown ||
			hostSoftware[0].MunkiResult != MunkiResultInstallIndicated {
			t.Fatalf("specific nopkg host detail = %+v, want same independent facts", hostSoftware)
		}
	})

	t.Run("latest all non-nopkg remains eligible", func(t *testing.T) {
		fixture := newDeploymentStoreFixture(t)
		fixture.followLatest(t)
		hosts := fixture.listInstalledOldHost(t)
		if len(hosts) != 1 || hosts[0].Status != StatusInstalled ||
			hosts[0].InstalledVersion != "1.2" {
			t.Fatalf("non-nopkg latest host = %+v, want installed 1.2", hosts)
		}
	})

	t.Run("latest all nopkg is unknown", func(t *testing.T) {
		fixture := newDeploymentStoreFixture(t)
		fixture.makeOnlyPackageNopkg(t)
		fixture.followLatest(t)
		hosts := fixture.listInstalledOldHost(t)
		if len(hosts) != 1 || hosts[0].Status != StatusUnknown {
			t.Fatalf("all-nopkg latest host = %+v, want unknown", hosts)
		}
	})

	t.Run("mixed latest is unknown", func(t *testing.T) {
		fixture := newDeploymentStoreFixture(t)
		fixture.followLatest(t)
		if _, err := fixture.db.Pool().Exec(fixture.ctx, `
INSERT INTO munki_packages (software_id, version, installer_type)
VALUES ($1, '2.0', 'nopkg')`, fixture.softwareID); err != nil {
			t.Fatalf("insert mixed nopkg package: %v", err)
		}
		hosts := fixture.listInstalledOldHost(t)
		if len(hosts) != 1 || hosts[0].Status != StatusUnknown {
			t.Fatalf("mixed latest host = %+v, want unknown", hosts)
		}
	})
}

func assertDeploymentFilter(
	t *testing.T,
	fixture deploymentStoreFixture,
	params DeploymentHostListParams,
	wantNames ...string,
) {
	t.Helper()
	hosts, count, err := fixture.store.ListDeploymentHosts(fixture.ctx, fixture.softwareID, params)
	if err != nil {
		t.Fatalf("filter deployment hosts: %v", err)
	}
	gotNames := make([]string, len(hosts))
	for i, host := range hosts {
		gotNames[i] = host.DisplayName
	}
	if count != len(wantNames) || !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("filtered hosts = %v count %d, want %v", gotNames, count, wantNames)
	}
}

func (f deploymentStoreFixture) listInstalledOldHost(t *testing.T) []DeploymentHost {
	t.Helper()
	hosts, _, err := f.store.ListDeploymentHosts(
		f.ctx,
		f.softwareID,
		DeploymentHostListParams{ListParams: dbutil.ListParams{PageSize: 1000, Q: "Installed Old"}},
	)
	if err != nil {
		t.Fatalf("ListDeploymentHosts Installed Old: %v", err)
	}
	return hosts
}

func (f deploymentStoreFixture) makeOnlyPackageNopkg(t *testing.T) {
	t.Helper()
	if _, err := f.db.Pool().Exec(f.ctx, `
UPDATE munki_packages
SET installer_type = 'nopkg', installer_object_id = NULL
WHERE id = $1`, f.packageID); err != nil {
		t.Fatalf("make package nopkg: %v", err)
	}
}

func (f deploymentStoreFixture) followLatest(t *testing.T) {
	t.Helper()
	if _, err := f.db.Pool().Exec(f.ctx, `
UPDATE munki_software_targets
SET package_selection = 'latest', pinned_package_id = NULL
WHERE software_id = $1 AND direction = 'include'`, f.softwareID); err != nil {
		t.Fatalf("follow latest packages: %v", err)
	}
}

type deploymentStoreFixture struct {
	ctx        context.Context
	db         *database.DB
	store      *Store
	softwareID int64
	packageID  int64
	labelID    int64
	hostIDs    map[string]int64
	now        time.Time
}

func newDeploymentStoreFixture(t *testing.T) deploymentStoreFixture {
	t.Helper()
	db, ctx := testdb.Open(t)
	fixture := deploymentStoreFixture{
		ctx:     ctx,
		db:      db,
		store:   &Store{db: db},
		hostIDs: make(map[string]int64),
		now:     time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
	}
	fixture.labelID = fixture.insertLabel(t, "Deployment Hosts")
	if err := db.Pool().QueryRow(ctx, `
INSERT INTO munki_software (
	name,
	installation_detector_bundle_identifier,
	installation_detector_expected_path,
	installation_detector_version_source
) VALUES ('Alpha Deployment', 'com.example.alpha', '/Applications/Alpha.app', 'bundle_short_version')
RETURNING id`).Scan(&fixture.softwareID); err != nil {
		t.Fatalf("insert software: %v", err)
	}
	var installerObjectID int64
	if err := db.Pool().QueryRow(ctx, `
INSERT INTO storage_objects (prefix, filename)
VALUES ('munki/packages', 'alpha-1.3.pkg') RETURNING id`).Scan(&installerObjectID); err != nil {
		t.Fatalf("insert package object: %v", err)
	}
	if err := db.Pool().QueryRow(ctx, `
INSERT INTO munki_packages (software_id, version, installer_type, installer_object_id)
VALUES ($1, '1.3', 'pkg', $2) RETURNING id`, fixture.softwareID, installerObjectID).Scan(&fixture.packageID); err != nil {
		t.Fatalf("insert package: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
INSERT INTO munki_software_targets (
	software_id, direction, position, label_id, actions, package_selection, pinned_package_id
) VALUES ($1, 'include', 0, $2, ARRAY['managed_installs']::munki_manifest_action[], 'specific', $3)`,
		fixture.softwareID, fixture.labelID, fixture.packageID); err != nil {
		t.Fatalf("insert target: %v", err)
	}

	installedOld := fixture.insertHost(t, "Installed Old")
	fixture.markSoftwareSnapshot(t, installedOld)
	fixture.insertApplication(t, installedOld, "1.2", "/Applications/Alpha.app")
	fixture.insertApplication(t, installedOld, "9.9", "/Applications/Elsewhere.app")
	fixture.makeMunkiCurrent(t, installedOld)
	fixture.insertMunkiItem(t, installedOld, true, "1.2", "1.3")

	munkiOnly := fixture.insertHost(t, "Munki Only")
	fixture.markSoftwareSnapshot(t, munkiOnly)
	fixture.makeMunkiCurrent(t, munkiOnly)
	fixture.insertMunkiItem(t, munkiOnly, true, "1.3", "")

	inventoryOnly := fixture.insertHost(t, "Inventory Only")
	fixture.markSoftwareSnapshot(t, inventoryOnly)
	fixture.insertApplication(t, inventoryOnly, "1.3", "/Applications/Alpha.app")
	fixture.makeMunkiFailed(t, inventoryOnly)

	emptySnapshot := fixture.insertHost(t, "Empty Snapshot")
	fixture.markSoftwareSnapshot(t, emptySnapshot)
	fixture.makeMunkiCurrent(t, emptySnapshot)

	fixture.insertHost(t, "No Snapshot")

	conflicting := fixture.insertHost(t, "Conflicting Versions")
	fixture.markSoftwareSnapshot(t, conflicting)
	fixture.insertApplication(t, conflicting, "2.0", "/Applications/Alpha.app")
	fixture.insertApplication(t, conflicting, "2.1", "/Applications/Alpha.app")

	return fixture
}

func (f deploymentStoreFixture) insertHost(t *testing.T, name string) int64 {
	t.Helper()
	var id int64
	if err := f.db.Pool().QueryRow(f.ctx, `
INSERT INTO hosts (hardware_uuid, display_name, hardware_serial)
VALUES ($1, $2, $3) RETURNING id`, "deployment-"+name, name, "serial-"+name).Scan(&id); err != nil {
		t.Fatalf("insert host %q: %v", name, err)
	}
	if _, err := f.db.Pool().Exec(
		f.ctx,
		`INSERT INTO label_membership (label_id, host_id) VALUES ($1, $2)`,
		f.labelID,
		id,
	); err != nil {
		t.Fatalf("insert host label %q: %v", name, err)
	}
	f.hostIDs[name] = id
	return id
}

func (f deploymentStoreFixture) insertLabel(t *testing.T, name string) int64 {
	t.Helper()
	var id int64
	if err := f.db.Pool().QueryRow(f.ctx, `
INSERT INTO labels (name, label_type, label_membership_type)
VALUES ($1, 'regular', 'manual') RETURNING id`, name).Scan(&id); err != nil {
		t.Fatalf("insert label: %v", err)
	}
	return id
}

func (f deploymentStoreFixture) markSoftwareSnapshot(t *testing.T, hostID int64) {
	t.Helper()
	if _, err := f.db.Pool().Exec(
		f.ctx,
		`UPDATE hosts SET software_inventory_updated_at = $2 WHERE id = $1`,
		hostID,
		f.now,
	); err != nil {
		t.Fatalf("mark software snapshot: %v", err)
	}
}

func (f deploymentStoreFixture) insertApplication(
	t *testing.T,
	hostID int64,
	version string,
	path string,
) {
	t.Helper()
	var softwareID int64
	if err := f.db.Pool().QueryRow(f.ctx, `
WITH title AS (
	INSERT INTO software_titles (name, source, bundle_identifier)
	VALUES ('Alpha', 'apps', 'com.example.alpha')
	ON CONFLICT (bundle_identifier, source, extension_for)
	WHERE bundle_identifier <> ''
	DO UPDATE SET updated_at = now()
	RETURNING id
)
INSERT INTO software (title_id, name, version, source, bundle_identifier)
SELECT id, 'Alpha', $1, 'apps', 'com.example.alpha' FROM title
ON CONFLICT (
	title_id, version, source, bundle_identifier, extension_id, extension_for, vendor, arch, release
) DO UPDATE SET updated_at = now()
RETURNING id`, version).Scan(&softwareID); err != nil {
		t.Fatalf("insert application %q: %v", version, err)
	}
	if _, err := f.db.Pool().Exec(f.ctx, `
INSERT INTO host_software (host_id, software_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING`, hostID, softwareID); err != nil {
		t.Fatalf("insert host application %q: %v", version, err)
	}
	if _, err := f.db.Pool().Exec(f.ctx, `
INSERT INTO host_software_installed_paths (
	host_id, software_id, installed_path, bundle_short_version, bundle_version
) VALUES ($1, $2, $3, $4, $4)
ON CONFLICT (host_id, software_id, installed_path) DO UPDATE SET
	bundle_short_version = EXCLUDED.bundle_short_version,
	bundle_version = EXCLUDED.bundle_version`, hostID, softwareID, path, version); err != nil {
		t.Fatalf("insert application path %q: %v", path, err)
	}
}

func (f deploymentStoreFixture) makeMunkiCurrent(t *testing.T, hostID int64) {
	t.Helper()
	if _, err := f.db.Pool().Exec(
		f.ctx,
		`INSERT INTO host_heartbeats (host_id, source, last_seen_at) VALUES ($1, 'munki', $2)`,
		hostID,
		f.now,
	); err != nil {
		t.Fatalf("touch host: %v", err)
	}
	if _, err := f.db.Pool().Exec(f.ctx, `
INSERT INTO munki_host_status (
	host_id, last_attempt_at, last_successful_at, collection_error, has_report
) VALUES ($1, $2, $2, '', TRUE)`, hostID, f.now); err != nil {
		t.Fatalf("insert host status: %v", err)
	}
}

func (f deploymentStoreFixture) makeMunkiFailed(t *testing.T, hostID int64) {
	t.Helper()
	if _, err := f.db.Pool().Exec(
		f.ctx,
		`INSERT INTO host_heartbeats (host_id, source, last_seen_at) VALUES ($1, 'munki', $2)`,
		hostID,
		f.now,
	); err != nil {
		t.Fatalf("touch failed host: %v", err)
	}
	if _, err := f.db.Pool().Exec(f.ctx, `
INSERT INTO munki_host_status (
	host_id, last_attempt_at, last_successful_at, collection_error, has_report
) VALUES ($1, $2, $2, 'munki_installs: unavailable', TRUE)`, hostID, f.now); err != nil {
		t.Fatalf("insert failed host status: %v", err)
	}
}

func (f deploymentStoreFixture) insertMunkiItem(
	t *testing.T,
	hostID int64,
	installed bool,
	installedVersion string,
	targetVersion string,
) {
	t.Helper()
	if _, err := f.db.Pool().Exec(f.ctx, `
INSERT INTO munki_host_items (
	host_id, name, display_name, installed, installed_version, target_version
) VALUES ($1, 'Alpha Deployment', '', $2, $3, $4)`,
		hostID, installed, installedVersion, targetVersion); err != nil {
		t.Fatalf("insert item: %v", err)
	}
}
