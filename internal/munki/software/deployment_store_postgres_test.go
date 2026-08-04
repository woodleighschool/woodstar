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

func TestDeploymentProjectionCombinesResolvedAssignmentsAndMunkiInstalls(t *testing.T) { //nolint:cyclop // One deployment projection lifecycle shares one database fixture.
	fixture := newDeploymentStoreFixture(t)

	summary, err := fixture.store.GetDeployment(fixture.ctx, fixture.softwareID)
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	want := DeploymentSummary{
		AssignedCount:  5,
		ReportingCount: 3,
		InstalledCount: 2,
		Packages: []PackageDeployment{
			{Version: "1.0", InstalledCount: 1},
			{Version: "2.0", AssignedCount: 4, ReportingCount: 3, InstalledCount: 1},
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
	if hostCount != 5 || len(hosts) != 5 {
		t.Fatalf("assigned hosts = %d/%d, want 5", len(hosts), hostCount)
	}
	byName := make(map[string]DeploymentHost, len(hosts))
	for _, host := range hosts {
		byName[host.DisplayName] = host
	}
	installedOld := byName["Installed Old"]
	if installedOld.ReportState != ReportCurrent || !installedOld.Installed ||
		installedOld.InstalledVersion != "1.0" || installedOld.TargetVersion != "2.0" ||
		deploymentStatusValue(installedOld.Status) != StatusPending {
		t.Fatalf("installed old host = %+v, want current 1.0 to 2.0", installedOld)
	}
	if omitted := byName["Omitted"]; omitted.ReportState != ReportCurrent || omitted.Installed ||
		deploymentStatusValue(omitted.Status) != StatusNotInstalled {
		t.Fatalf("omitted host = %+v, want current and not installed", omitted)
	}
	if noContact := byName["No Contact"]; noContact.ReportState != ReportNotContacted {
		t.Fatalf("no-contact host = %+v, want not contacted", noContact)
	}
	if optional := byName["Installed Current"]; optional.Package.Strategy != PackageLatest ||
		!optional.Installed || optional.InstalledVersion != "2.0" || optional.TargetVersion != "" ||
		deploymentStatusValue(optional.Status) != StatusInstalled {
		t.Fatalf("optional host = %+v, want installed latest package", optional)
	}

	filtered, filteredCount, err := fixture.store.ListDeploymentHosts(
		fixture.ctx,
		fixture.softwareID,
		DeploymentHostListParams{
			ListParams: dbutil.ListParams{PageSize: 1000, Q: "serial-Installed Old"},
		},
	)
	if err != nil {
		t.Fatalf("search deployment hosts: %v", err)
	}
	if filteredCount != 1 || len(filtered) != 1 || filtered[0].DisplayName != "Installed Old" {
		t.Fatalf("filtered hosts = %+v count %d, want Installed Old", filtered, filteredCount)
	}

	pending := StatusPending
	filtered, filteredCount, err = fixture.store.ListDeploymentHosts(
		fixture.ctx,
		fixture.softwareID,
		DeploymentHostListParams{
			ListParams: dbutil.ListParams{PageSize: 1000},
			Status:     &pending,
		},
	)
	if err != nil {
		t.Fatalf("filter pending hosts: %v", err)
	}
	if filteredCount != 1 || len(filtered) != 1 || filtered[0].DisplayName != "Installed Old" {
		t.Fatalf("pending hosts = %+v count %d, want Installed Old", filtered, filteredCount)
	}

	optionalAction := ActionOptionalInstalls
	filtered, filteredCount, err = fixture.store.ListDeploymentHosts(
		fixture.ctx,
		fixture.softwareID,
		DeploymentHostListParams{
			ListParams: dbutil.ListParams{PageSize: 1000},
			Action:     &optionalAction,
		},
	)
	if err != nil {
		t.Fatalf("filter optional assignments: %v", err)
	}
	if filteredCount != 2 || len(filtered) != 2 ||
		filtered[0].DisplayName != "Installed Current" ||
		filtered[1].DisplayName != "Optional No Contact" {
		t.Fatalf("optional hosts = %+v count %d, want current and no-contact hosts", filtered, filteredCount)
	}

	rows, count, err := fixture.store.ListWithDeployment(
		fixture.ctx,
		dbutil.ListParams{PageSize: 1},
	)
	if err != nil {
		t.Fatalf("ListWithDeployment: %v", err)
	}
	if count != 1 || len(rows) != 1 || !reflect.DeepEqual(rows[0].Deployment, want) {
		t.Fatalf("software page = %+v count %d, want deployment summary %+v", rows, count, want)
	}
}

type deploymentStoreFixture struct {
	ctx        context.Context
	db         *database.DB
	store      *Store
	softwareID int64
	required   int64
	optional   int64
	exclude    int64
	now        time.Time
}

func newDeploymentStoreFixture(t *testing.T) deploymentStoreFixture {
	t.Helper()
	db, ctx := testdb.Open(t)
	fixture := deploymentStoreFixture{
		ctx:   ctx,
		db:    db,
		store: &Store{db: db},
		now:   time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
	}
	fixture.required = fixture.insertLabel(t, "Deployment Required")
	fixture.optional = fixture.insertLabel(t, "Deployment Optional")
	fixture.exclude = fixture.insertLabel(t, "Deployment Excluded")
	fixture.softwareID = fixture.insertSoftware(t, "Alpha Deployment")

	var oldPackageID, currentPackageID int64
	if err := db.Pool().QueryRow(ctx, `
INSERT INTO munki_packages (software_id, version, installer_type)
VALUES ($1, '1.0', 'nopkg') RETURNING id`, fixture.softwareID).Scan(&oldPackageID); err != nil {
		t.Fatalf("insert old package: %v", err)
	}
	if err := db.Pool().QueryRow(ctx, `
INSERT INTO munki_packages (software_id, version, installer_type)
VALUES ($1, '2.0', 'nopkg') RETURNING id`, fixture.softwareID).Scan(&currentPackageID); err != nil {
		t.Fatalf("insert current package: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
INSERT INTO munki_software_targets (
	software_id, direction, position, label_id, actions, package_selection, pinned_package_id
) VALUES
	($1, 'include', 0, $2, ARRAY['managed_installs']::munki_manifest_action[], 'specific', $5),
	($1, 'include', 1, $3, ARRAY['optional_installs']::munki_manifest_action[], 'latest', NULL),
	($1, 'exclude', 0, $4, NULL, NULL, NULL)`, fixture.softwareID, fixture.required, fixture.optional, fixture.exclude, currentPackageID); err != nil {
		t.Fatalf("insert targets: %v", err)
	}

	installedOld := fixture.insertHost(t, "Installed Old", fixture.required)
	fixture.makeCurrent(t, installedOld)
	fixture.insertItem(t, installedOld, true, "1.0", "2.0")

	omitted := fixture.insertHost(t, "Omitted", fixture.required)
	fixture.makeCurrent(t, omitted)

	fixture.insertHost(t, "No Contact", fixture.required)

	installedCurrent := fixture.insertHost(t, "Installed Current", fixture.optional)
	fixture.makeCurrent(t, installedCurrent)
	fixture.insertItem(t, installedCurrent, true, "2.0", "")

	fixture.insertHost(t, "Optional No Contact", fixture.optional)

	excluded := fixture.insertHost(t, "Excluded", fixture.required, fixture.exclude)
	fixture.makeCurrent(t, excluded)
	fixture.insertItem(t, excluded, true, "2.0", "2.0")

	return fixture
}

func (f deploymentStoreFixture) insertHost(t *testing.T, name string, labelIDs ...int64) int64 {
	t.Helper()
	var id int64
	if err := f.db.Pool().QueryRow(f.ctx, `
INSERT INTO hosts (hardware_uuid, display_name, hardware_serial)
VALUES ($1, $2, $3) RETURNING id`, "deployment-"+name, name, "serial-"+name).Scan(&id); err != nil {
		t.Fatalf("insert host %q: %v", name, err)
	}
	for _, labelID := range labelIDs {
		if _, err := f.db.Pool().Exec(
			f.ctx,
			`INSERT INTO label_membership (label_id, host_id) VALUES ($1, $2)`,
			labelID,
			id,
		); err != nil {
			t.Fatalf("insert host label %q: %v", name, err)
		}
	}
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

func (f deploymentStoreFixture) insertSoftware(t *testing.T, name string) int64 {
	t.Helper()
	var id int64
	if err := f.db.Pool().QueryRow(
		f.ctx,
		`INSERT INTO munki_software (name) VALUES ($1) RETURNING id`,
		name,
	).Scan(&id); err != nil {
		t.Fatalf("insert software: %v", err)
	}
	return id
}

func (f deploymentStoreFixture) makeCurrent(t *testing.T, hostID int64) {
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

func (f deploymentStoreFixture) insertItem(
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
) VALUES ($1, 'Alpha Deployment', '', $2, $3, $4)`, hostID, installed, installedVersion, targetVersion); err != nil {
		t.Fatalf("insert item: %v", err)
	}
}
