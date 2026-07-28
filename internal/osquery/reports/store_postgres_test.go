//go:build postgres

package reports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestListIncludesTargets(t *testing.T) {
	store, labelStore, _, ctx := newPostgresReportStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Report A")
	labelB := createManualLabel(t, ctx, labelStore, "Report B")
	labelC := createManualLabel(t, ctx, labelStore, "Report C")

	if _, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Targeted report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{labelB.ID, labelA.ID}, []int64{labelC.ID}),
	}}); err != nil {
		t.Fatalf("create report: %v", err)
	}

	got, count, err := store.List(ctx, ReportListParams{})
	if err != nil {
		t.Fatalf("list reports: %v", err)
	}
	if count != 1 || len(got) != 1 {
		t.Fatalf("List returned count=%d len=%d, want one report", count, len(got))
	}
	assertTargets(t, got[0].Targets, reportTargets([]int64{labelB.ID, labelA.ID}, []int64{labelC.ID}))
}

func TestUpdateReplacesTargets(t *testing.T) {
	store, labelStore, _, ctx := newPostgresReportStore(t)
	first := createManualLabel(t, ctx, labelStore, "Report first")
	second := createManualLabel(t, ctx, labelStore, "Report second")
	third := createManualLabel(t, ctx, labelStore, "Report third")

	report, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Replacement report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{first.ID, second.ID}, []int64{third.ID}),
	}})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}

	updated, err := store.Update(ctx, report.ID, ReportMutation{
		Name:             "Replacement report",
		Query:            "select 2;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{third.ID}, []int64{first.ID}),
	})
	if err != nil {
		t.Fatalf("update report: %v", err)
	}
	assertTargets(t, updated.Targets, reportTargets([]int64{third.ID}, []int64{first.ID}))

	got, err := store.GetByID(ctx, report.ID)
	if err != nil {
		t.Fatalf("get updated report: %v", err)
	}
	assertTargets(t, got.Targets, reportTargets([]int64{third.ID}, []int64{first.ID}))
}

func TestScheduledForHostUsesTargetRows(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-target-host")
	matching := createManualLabel(t, ctx, labelStore, "Report match")
	other := createManualLabel(t, ctx, labelStore, "Report other")
	excluded := createManualLabel(t, ctx, labelStore, "Report excluded")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Matching scheduled report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{matching.ID}, nil),
	}}); err != nil {
		t.Fatalf("create matching report: %v", err)
	}
	if _, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Nonmatching scheduled report",
		Query:            "select 2;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{other.ID}, nil),
	}}); err != nil {
		t.Fatalf("create nonmatching report: %v", err)
	}
	if _, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Excluded scheduled report",
		Query:            "select 3;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{matching.ID}, []int64{excluded.ID}),
	}}); err != nil {
		t.Fatalf("create excluded report: %v", err)
	}

	got, err := store.ScheduledForHost(ctx, host)
	if err != nil {
		t.Fatalf("scheduled for host: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Matching scheduled report" {
		t.Fatalf("ScheduledForHost returned %+v, want only matching report", got)
	}
}

func TestScheduledForHostRequiresIncludeTarget(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-requires-include-host")
	excluded := createManualLabel(t, ctx, labelStore, "Report requires include excluded")
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Exclude-only scheduled report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets(nil, []int64{excluded.ID}),
	}}); err != nil {
		t.Fatalf("create exclude-only report: %v", err)
	}

	got, err := store.ScheduledForHost(ctx, host)
	if err != nil {
		t.Fatalf("scheduled for host: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ScheduledForHost returned %+v, want no reports", got)
	}
}

func TestCreateReportWithMissingLabelReturnsNotFound(t *testing.T) {
	store, _, _, ctx := newPostgresReportStore(t)

	_, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Missing label target",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{999_999}, nil),
	}})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Create error = %v, want ErrNotFound", err)
	}
}

func TestScheduledForHostUsesScheduleState(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-applicable-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)

	if _, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:              "Matching scheduled report",
		Query:             "select 1;",
		MinOsqueryVersion: new("5.0.0"),
		ScheduleInterval:  60,
		Targets:           reportTargets([]int64{allHostsID}, nil),
	}}); err != nil {
		t.Fatalf("create matching report: %v", err)
	}
	if _, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Unscheduled report",
		Query:            "select 2;",
		ScheduleInterval: 0,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}}); err != nil {
		t.Fatalf("create unscheduled report: %v", err)
	}
	if _, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:              "Version-gated scheduled report",
		Query:             "select 4;",
		MinOsqueryVersion: new("6.0.0"),
		ScheduleInterval:  60,
		Targets:           reportTargets([]int64{allHostsID}, nil),
	}}); err != nil {
		t.Fatalf("create version-gated report: %v", err)
	}

	got, err := store.ScheduledForHost(ctx, host)
	if err != nil {
		t.Fatalf("scheduled for host: %v", err)
	}
	if len(got) != 2 || got[0].Name != "Matching scheduled report" || got[1].Name != "Version-gated scheduled report" {
		t.Fatalf("ScheduledForHost returned %+v, want scheduled reports", got)
	}
	if got[1].MinOsqueryVersion == nil || *got[1].MinOsqueryVersion != "6.0.0" {
		t.Fatalf("ScheduledForHost min version = %v, want preserved schedule metadata", got[1].MinOsqueryVersion)
	}
}

func TestHostReportsIncludeLatestHostState(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	fetchedAt := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)

	reportWithRows, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Report with rows",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}})
	if err != nil {
		t.Fatalf("create report with rows: %v", err)
	}
	reportEmpty, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Report empty",
		Query:            "select name from missing_apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}})
	if err != nil {
		t.Fatalf("create empty report: %v", err)
	}
	if err := store.OverwriteResults(ctx, reportWithRows.ID, testQueryHash(reportWithRows.Query), host.ID, []map[string]string{
		{"name": "Alpha"},
		{"name": "Bravo"},
	}, fetchedAt); err != nil {
		t.Fatalf("overwrite report rows: %v", err)
	}
	if err := store.OverwriteResults(
		ctx,
		reportEmpty.ID,
		testQueryHash(reportEmpty.Query),
		host.ID,
		nil,
		fetchedAt.Add(time.Minute),
	); err != nil {
		t.Fatalf("overwrite empty report: %v", err)
	}

	got, err := store.HostReports(ctx, host)
	if err != nil {
		t.Fatalf("host reports: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("HostReports returned %d reports, want 2: %+v", len(got), got)
	}
	byID := make(map[int64]HostReport, len(got))
	for _, report := range got {
		byID[report.ReportID] = report
	}

	withRows := byID[reportWithRows.ID]
	if withRows.HostResultCount != 2 {
		t.Fatalf("HostResultCount = %d, want 2", withRows.HostResultCount)
	}
	if withRows.LastFetched == nil || !withRows.LastFetched.Equal(fetchedAt) {
		t.Fatalf("LastFetched = %v, want %s", withRows.LastFetched, fetchedAt)
	}
	if withRows.FirstResult["name"] != "Bravo" {
		t.Fatalf("FirstResult = %#v, want latest row", withRows.FirstResult)
	}

	empty := byID[reportEmpty.ID]
	if empty.HostResultCount != 0 {
		t.Fatalf("empty HostResultCount = %d, want 0", empty.HostResultCount)
	}
	wantEmptyFetched := fetchedAt.Add(time.Minute)
	if empty.LastFetched == nil || !empty.LastFetched.Equal(wantEmptyFetched) {
		t.Fatalf("empty LastFetched = %v, want %s", empty.LastFetched, wantEmptyFetched)
	}
	if empty.FirstResult != nil {
		t.Fatalf("empty FirstResult = %#v, want nil", empty.FirstResult)
	}
}

func TestOverwriteResultsReplacesHostSnapshot(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	host := enrollTestHost(t, ctx, hostStore, "report-overwrite-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	report, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Overwrite report",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}

	firstFetchedAt := time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC)
	if err := store.OverwriteResults(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Alpha"},
		{"name": "Bravo"},
	}, firstFetchedAt); err != nil {
		t.Fatalf("overwrite first snapshot: %v", err)
	}
	secondFetchedAt := firstFetchedAt.Add(time.Hour)
	if err := store.OverwriteResults(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Charlie"},
	}, secondFetchedAt); err != nil {
		t.Fatalf("overwrite second snapshot: %v", err)
	}

	got, err := store.Results(ctx, report.ID)
	if err != nil {
		t.Fatalf("report results: %v", err)
	}
	if len(got) != 1 || got[0].Columns["name"] != "Charlie" {
		t.Fatalf("Results = %+v, want only replacement row", got)
	}

	emptyFetchedAt := secondFetchedAt.Add(time.Hour)
	if err := store.OverwriteResults(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		host.ID,
		nil,
		emptyFetchedAt,
	); err != nil {
		t.Fatalf("overwrite empty snapshot: %v", err)
	}
	got, err = store.Results(ctx, report.ID)
	if err != nil {
		t.Fatalf("report results after empty snapshot: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Results after empty snapshot = %+v, want no data rows", got)
	}
}

func TestUpdateInvalidatesResultsOnlyWhenQueryChanges(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	host := enrollTestHost(t, ctx, hostStore, "report-query-change-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	targets := reportTargets([]int64{allHostsID}, nil)
	report, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Query change report",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          targets,
	}})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	fetchedAt := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	if err := store.OverwriteResults(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Alpha"},
	}, fetchedAt); err != nil {
		t.Fatalf("store report result: %v", err)
	}

	metadataUpdated, err := store.Update(ctx, report.ID, ReportMutation{
		Name:             "Renamed report",
		Description:      "Non-semantic edit",
		Query:            " select name from apps; ",
		ScheduleInterval: 120,
		Targets:          targets,
	})
	if err != nil {
		t.Fatalf("update report metadata: %v", err)
	}
	if testQueryHash(metadataUpdated.Query) != testQueryHash(report.Query) {
		t.Fatal("metadata edit changed the normalized query hash")
	}
	got, err := store.Results(ctx, report.ID)
	if err != nil {
		t.Fatalf("results after metadata edit: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("results after metadata edit = %+v, want saved result", got)
	}

	queryUpdated, err := store.Update(ctx, report.ID, ReportMutation{
		Name:             "Renamed report",
		Description:      "Non-semantic edit",
		Query:            "select name from users;",
		ScheduleInterval: 120,
		Targets:          targets,
	})
	if err != nil {
		t.Fatalf("update report query: %v", err)
	}
	if testQueryHash(queryUpdated.Query) == testQueryHash(report.Query) {
		t.Fatal("changed query retained its previous query hash")
	}
	got, err = store.Results(ctx, report.ID)
	if err != nil {
		t.Fatalf("results after query edit: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("results after query edit = %+v, want no stale results", got)
	}
	if err := store.OverwriteResults(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		host.ID,
		[]map[string]string{{"name": "Obsolete"}},
		fetchedAt.Add(time.Minute),
	); err != nil {
		t.Fatalf("store obsolete report result: %v", err)
	}
	got, err = store.Results(ctx, report.ID)
	if err != nil {
		t.Fatalf("results after obsolete snapshot: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("results after obsolete snapshot = %+v, want none", got)
	}
	if err := store.OverwriteResults(
		ctx,
		report.ID,
		testQueryHash(queryUpdated.Query),
		host.ID,
		[]map[string]string{{"name": "Current"}},
		fetchedAt.Add(2*time.Minute),
	); err != nil {
		t.Fatalf("store current report result: %v", err)
	}
	got, err = store.Results(ctx, report.ID)
	if err != nil {
		t.Fatalf("results after current snapshot: %v", err)
	}
	if len(got) != 1 || got[0].Columns["name"] != "Current" {
		t.Fatalf("results after current snapshot = %+v, want Current", got)
	}
}

func TestResultsHiddenWhenHostLeavesScope(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	host := enrollTestHost(t, ctx, hostStore, "report-excluded-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	excluded := createManualLabel(t, ctx, labelStore, "Report excluded results")
	report, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Scoped report",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, []int64{excluded.ID}),
	}})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	fetchedAt := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	if err := store.OverwriteResults(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Alpha"},
	}, fetchedAt); err != nil {
		t.Fatalf("store report result: %v", err)
	}

	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("exclude report host: %v", err)
	}
	assertReportResultCount(t, ctx, store, report.ID, host.ID, 1)
	got, err := store.Results(ctx, report.ID)
	if err != nil {
		t.Fatalf("results after host exclusion: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("results after host exclusion = %+v, want none", got)
	}

	if err := store.OverwriteResults(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Late result"},
	}, fetchedAt.Add(time.Minute)); err != nil {
		t.Fatalf("store late unassigned result: %v", err)
	}
	assertReportResultCount(t, ctx, store, report.ID, host.ID, 1)
	var persistedName string
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT data->>'name'
		FROM osquery_report_results
		WHERE report_id = $1 AND host_id = $2`,
		report.ID,
		host.ID,
	).Scan(&persistedName); err != nil {
		t.Fatalf("read retained report result: %v", err)
	}
	if persistedName != "Alpha" {
		t.Fatalf("retained report result = %q, want Alpha", persistedName)
	}
}

func TestUpdatePrunesResultsOutsideNewTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	retainedHost := enrollTestHost(t, ctx, hostStore, "report-retained-host")
	removedHost := enrollTestHost(t, ctx, hostStore, "report-removed-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	retainedLabel := createManualLabel(t, ctx, labelStore, "Report retained")
	if err := labelStore.SetMembership(ctx, retainedLabel.ID, retainedHost.ID, true); err != nil {
		t.Fatalf("retain report host: %v", err)
	}
	report, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Retargeted report",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	fetchedAt := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	for _, hostID := range []int64{retainedHost.ID, removedHost.ID} {
		if err := store.OverwriteResults(ctx, report.ID, testQueryHash(report.Query), hostID, []map[string]string{
			{"name": "Alpha"},
		}, fetchedAt); err != nil {
			t.Fatalf("store host %d report result: %v", hostID, err)
		}
	}

	if _, err := store.Update(ctx, report.ID, ReportMutation{
		Name:             "Retargeted report",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{retainedLabel.ID}, nil),
	}); err != nil {
		t.Fatalf("retarget report: %v", err)
	}

	rows, err := store.db.Pool().Query(ctx, `
		SELECT DISTINCT host_id
		FROM osquery_report_results
		WHERE report_id = $1
		ORDER BY host_id`,
		report.ID,
	)
	if err != nil {
		t.Fatalf("list persisted report hosts: %v", err)
	}
	hostIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		t.Fatalf("collect persisted report hosts: %v", err)
	}
	if len(hostIDs) != 1 || hostIDs[0] != retainedHost.ID {
		t.Fatalf("persisted report host IDs = %v, want [%d]", hostIDs, retainedHost.ID)
	}
}

func assertReportResultCount(
	t *testing.T,
	ctx context.Context,
	store *Store,
	reportID int64,
	hostID int64,
	want int,
) {
	t.Helper()
	var got int
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT count(*)
		FROM osquery_report_results
		WHERE report_id = $1 AND host_id = $2`,
		reportID,
		hostID,
	).Scan(&got); err != nil {
		t.Fatalf("count persisted report results: %v", err)
	}
	if got != want {
		t.Fatalf("persisted report results = %d, want %d", got, want)
	}
}

func newPostgresReportStore(t *testing.T) (*Store, *labels.Store, *hosts.Store, context.Context) {
	t.Helper()
	database, ctx := testdb.Open(t)
	return NewStore(database), labels.NewStore(database), hosts.NewStore(database), ctx
}

func createManualLabel(t *testing.T, ctx context.Context, store *labels.Store, name string) *labels.Label {
	t.Helper()
	label, err := store.Create(ctx, labels.LabelMutation{
		Name:                name,
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label %q: %v", name, err)
	}
	return label
}

func enrollTestHost(t *testing.T, ctx context.Context, store *hosts.Store, hardwareUUID string) *hosts.Host {
	t.Helper()
	host, err := store.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: hardwareUUID},
		OrbitNodeKey: hardwareUUID + "-node-key",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	return host
}

func enrollTestHostDetail(
	t *testing.T,
	ctx context.Context,
	store *hosts.Store,
	hardwareUUID string,
) *hosts.Host {
	t.Helper()
	host, err := store.UpsertOnOsqueryEnroll(ctx, hosts.InventoryUpdate{
		Hardware:       hosts.HostHardware{UUID: hardwareUUID},
		OsqueryNodeKey: hardwareUUID + "-node-key",
		Agents:         hosts.HostAgents{Osquery: hosts.HostOsqueryAgent{Version: "5.22.1"}},
	})
	if err != nil {
		t.Fatalf("enroll osquery host: %v", err)
	}
	return host
}

func allHostsLabelID(t *testing.T, ctx context.Context, store *labels.Store) int64 {
	t.Helper()
	rows, _, err := store.List(ctx, labels.LabelListParams{})
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

func testQueryHash(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}

func reportTargets(includeIDs, excludeIDs []int64) ReportTargets {
	return ReportTargets{
		Include: labelRefs(includeIDs...),
		Exclude: labelRefs(excludeIDs...),
	}
}

func labelRefs(labelIDs ...int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(labelIDs))
	for i, labelID := range labelIDs {
		refs[i] = targeting.LabelRef{LabelID: labelID}
	}
	return refs
}

func assertTargets(t *testing.T, got ReportTargets, want ReportTargets) {
	t.Helper()
	if len(got.Include) != len(want.Include) || len(got.Exclude) != len(want.Exclude) {
		t.Fatalf("targets = %#v, want %#v", got, want)
	}
	for i := range want.Include {
		if got.Include[i] != want.Include[i] {
			t.Fatalf("targets = %#v, want %#v", got, want)
		}
	}
	for i := range want.Exclude {
		if got.Exclude[i] != want.Exclude[i] {
			t.Fatalf("targets = %#v, want %#v", got, want)
		}
	}
}
