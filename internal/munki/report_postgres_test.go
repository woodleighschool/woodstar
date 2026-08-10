//go:build postgres

package munki_test

import (
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/munki"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestSoftwareReportUsesManagedInstallsAndMunkiEvaluation(t *testing.T) { //nolint:funlen // One report lifecycle.
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(db)

	hostNames := []string{"Alpha", "Bravo", "Charlie"}
	hostsByName := make(map[string]*hosts.Host, len(hostNames))
	for _, name := range hostNames {
		host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
			ComputerName: name,
			Hardware: hosts.HostHardware{
				UUID:   "software-report-" + name,
				Serial: "REPORT-" + name,
			},
			OrbitNodeKey: "software-report-" + name + "-orbit",
		})
		if err != nil {
			t.Fatalf("enroll %s: %v", name, err)
		}
		hostsByName[name] = host
	}

	allHostsID := allHostsLabelID(t, ctx, labelStore)
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "ReportApp"})
	if err != nil {
		t.Fatalf("create report software: %v", err)
	}
	createMunkiPackage(t, ctx, stores, title.ID, title.Name, "2.0")
	replaceTargets(t, ctx, stores, title, []munkisoftware.Include{
		includeTarget(allHostsID, munkisoftware.ActionManagedInstalls),
	})

	collector := munki.NewDetailIngestor(stores.hoststate)
	evaluatedAt := time.Date(2026, 8, 9, 10, 11, 12, 0, time.UTC)
	info := munki.QueryResult{Present: true, Successful: true, Rows: []map[string]string{{
		"end_time": evaluatedAt.Format("2006-01-02 15:04:05 -0700"),
	}}}
	if err := collector.IngestCollection(ctx, hostsByName["Alpha"].ID, munki.Collection{
		Info: info,
		Installs: munki.QueryResult{Present: true, Successful: true, Rows: []map[string]string{{
			"name":              title.Name,
			"installed":         "true",
			"installed_version": "2.0",
		}}},
	}); err != nil {
		t.Fatalf("ingest installed report: %v", err)
	}
	if err := collector.IngestCollection(ctx, hostsByName["Bravo"].ID, munki.Collection{
		Info: info,
		Installs: munki.QueryResult{Present: true, Successful: true, Rows: []map[string]string{{
			"name":               title.Name,
			"installed":          "false",
			"version_to_install": "2.0",
		}}},
	}); err != nil {
		t.Fatalf("ingest pending report: %v", err)
	}

	report, count, err := stores.software.ListReportHosts(
		ctx,
		title.ID,
		munkisoftware.SoftwareReportHostListParams{},
	)
	if err != nil {
		t.Fatalf("list report: %v", err)
	}
	if count != 3 || len(report) != 3 {
		t.Fatalf("report = %+v count %d, want three expected hosts", report, count)
	}
	assertSoftwareReportHost(
		t,
		report[0],
		"Alpha",
		munkisoftware.SoftwareReportStatusInstalled,
		"",
		&evaluatedAt,
	)
	assertSoftwareReportHost(
		t,
		report[1],
		"Bravo",
		munkisoftware.SoftwareReportStatusPending,
		"2.0",
		&evaluatedAt,
	)
	assertSoftwareReportHost(t, report[2], "Charlie", "", "", nil)

	title, err = stores.software.GetByID(ctx, title.ID)
	if err != nil {
		t.Fatalf("get report software: %v", err)
	}
	if title.InstalledHostCount != 1 || title.ExpectedHostCount != 3 {
		t.Fatalf(
			"software counts = %d / %d, want 1 / 3",
			title.InstalledHostCount,
			title.ExpectedHostCount,
		)
	}

	installed, count, err := stores.software.ListReportHosts(ctx, title.ID, munkisoftware.SoftwareReportHostListParams{
		Statuses: []munkisoftware.SoftwareReportStatus{munkisoftware.SoftwareReportStatusInstalled},
	})
	if err != nil {
		t.Fatalf("filter installed report: %v", err)
	}
	if count != 1 || len(installed) != 1 || installed[0].HostName != "Alpha" {
		t.Fatalf("installed report = %+v count %d, want Alpha", installed, count)
	}
	pending, count, err := stores.software.ListReportHosts(ctx, title.ID, munkisoftware.SoftwareReportHostListParams{
		ListParams: listing.Params{Q: "REPORT-Bravo"},
		Statuses:   []munkisoftware.SoftwareReportStatus{munkisoftware.SoftwareReportStatusPending},
	})
	if err != nil {
		t.Fatalf("search pending report: %v", err)
	}
	if count != 1 || len(pending) != 1 || pending[0].HostName != "Bravo" {
		t.Fatalf("pending report = %+v count %d, want Bravo", pending, count)
	}

	updateOnly, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "UpdateOnlyApp"})
	if err != nil {
		t.Fatalf("create update-only software: %v", err)
	}
	createMunkiPackage(t, ctx, stores, updateOnly.ID, updateOnly.Name, "1.0")
	replaceTargets(t, ctx, stores, updateOnly, []munkisoftware.Include{
		includeTarget(allHostsID, munkisoftware.ActionManagedUpdates),
	})
	updateOnly, err = stores.software.GetByID(ctx, updateOnly.ID)
	if err != nil {
		t.Fatalf("get update-only software: %v", err)
	}
	if updateOnly.InstalledHostCount != 0 || updateOnly.ExpectedHostCount != 0 {
		t.Fatalf(
			"update-only counts = %d / %d, want 0 / 0",
			updateOnly.InstalledHostCount,
			updateOnly.ExpectedHostCount,
		)
	}
	updateReport, count, err := stores.software.ListReportHosts(
		ctx,
		updateOnly.ID,
		munkisoftware.SoftwareReportHostListParams{},
	)
	if err != nil {
		t.Fatalf("list update-only report: %v", err)
	}
	if count != 0 || len(updateReport) != 0 {
		t.Fatalf("update-only report = %+v count %d, want empty", updateReport, count)
	}
}

func assertSoftwareReportHost(
	t *testing.T,
	got munkisoftware.SoftwareReportHost,
	wantName string,
	wantStatus munkisoftware.SoftwareReportStatus,
	wantTarget string,
	wantEvaluatedAt *time.Time,
) {
	t.Helper()
	if got.HostName != wantName || got.TargetVersion != wantTarget {
		t.Fatalf("report host = %+v, want %s target %q", got, wantName, wantTarget)
	}
	if wantStatus == "" {
		if got.Status != nil {
			t.Fatalf("%s status = %q, want none", wantName, *got.Status)
		}
	} else if got.Status == nil || *got.Status != wantStatus {
		t.Fatalf("%s status = %v, want %q", wantName, got.Status, wantStatus)
	}
	if wantEvaluatedAt == nil {
		if got.EvaluatedAt != nil {
			t.Fatalf("%s evaluated_at = %v, want none", wantName, got.EvaluatedAt)
		}
	} else if got.EvaluatedAt == nil || !got.EvaluatedAt.Equal(*wantEvaluatedAt) {
		t.Fatalf("%s evaluated_at = %v, want %v", wantName, got.EvaluatedAt, wantEvaluatedAt)
	}
}
