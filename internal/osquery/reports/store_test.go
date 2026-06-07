package reports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestListIncludesTargets(t *testing.T) {
	store, labelStore, _, ctx := newIntegrationReportStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Report A")
	labelB := createManualLabel(t, ctx, labelStore, "Report B")
	labelC := createManualLabel(t, ctx, labelStore, "Report C")

	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Targeted report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{labelB.ID, labelA.ID}, []int64{labelC.ID}),
	}); err != nil {
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
	store, labelStore, _, ctx := newIntegrationReportStore(t)
	first := createManualLabel(t, ctx, labelStore, "Report first")
	second := createManualLabel(t, ctx, labelStore, "Report second")
	third := createManualLabel(t, ctx, labelStore, "Report third")

	report, err := store.Create(ctx, ReportMutation{
		Name:             "Replacement report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{first.ID, second.ID}, []int64{third.ID}),
	})
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
	store, labelStore, hostStore, ctx := newIntegrationReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-target-host", "5.22.1")
	matching := createManualLabel(t, ctx, labelStore, "Report match")
	other := createManualLabel(t, ctx, labelStore, "Report other")
	excluded := createManualLabel(t, ctx, labelStore, "Report excluded")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Matching scheduled report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{matching.ID}, nil),
	}); err != nil {
		t.Fatalf("create matching report: %v", err)
	}
	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Nonmatching scheduled report",
		Query:            "select 2;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{other.ID}, nil),
	}); err != nil {
		t.Fatalf("create nonmatching report: %v", err)
	}
	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Excluded scheduled report",
		Query:            "select 3;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{matching.ID}, []int64{excluded.ID}),
	}); err != nil {
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
	store, labelStore, hostStore, ctx := newIntegrationReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-requires-include-host", "5.22.1")
	excluded := createManualLabel(t, ctx, labelStore, "Report requires include excluded")
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Exclude-only scheduled report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets(nil, []int64{excluded.ID}),
	}); err != nil {
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
	store, _, _, ctx := newIntegrationReportStore(t)

	_, err := store.Create(ctx, ReportMutation{
		Name:             "Missing label target",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{999_999}, nil),
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Create error = %v, want ErrNotFound", err)
	}
}

func TestCreateReportRejectsIncludeExcludeTargetOverlap(t *testing.T) {
	store, labelStore, _, ctx := newIntegrationReportStore(t)
	label := createManualLabel(t, ctx, labelStore, "Report Overlap")

	_, err := store.Create(ctx, ReportMutation{
		Name:             "Overlapping report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{label.ID}, []int64{label.ID}),
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Create error = %v, want ErrInvalidInput", err)
	}
}

func TestScheduledForHostUsesScheduleState(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-applicable-host", "5.22.1")
	allHostsID := allHostsLabelID(t, ctx, labelStore)

	if _, err := store.Create(ctx, ReportMutation{
		Name:              "Matching scheduled report",
		Query:             "select 1;",
		MinOsqueryVersion: new("5.0.0"),
		ScheduleInterval:  60,
		Targets:           reportTargets([]int64{allHostsID}, nil),
	}); err != nil {
		t.Fatalf("create matching report: %v", err)
	}
	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Unscheduled report",
		Query:            "select 2;",
		ScheduleInterval: 0,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}); err != nil {
		t.Fatalf("create unscheduled report: %v", err)
	}
	if _, err := store.Create(ctx, ReportMutation{
		Name:              "Version-gated scheduled report",
		Query:             "select 4;",
		MinOsqueryVersion: new("6.0.0"),
		ScheduleInterval:  60,
		Targets:           reportTargets([]int64{allHostsID}, nil),
	}); err != nil {
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
	store, labelStore, hostStore, ctx := newIntegrationReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-host", "5.22.1")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	fetchedAt := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)

	reportWithRows, err := store.Create(ctx, ReportMutation{
		Name:             "Report with rows",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	})
	if err != nil {
		t.Fatalf("create report with rows: %v", err)
	}
	reportEmpty, err := store.Create(ctx, ReportMutation{
		Name:             "Report empty",
		Query:            "select name from missing_apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	})
	if err != nil {
		t.Fatalf("create empty report: %v", err)
	}
	if err := store.OverwriteResults(ctx, reportWithRows.ID, host.ID, []map[string]string{
		{"name": "Alpha"},
		{"name": "Bravo"},
	}, fetchedAt); err != nil {
		t.Fatalf("overwrite report rows: %v", err)
	}
	if err := store.OverwriteResults(ctx, reportEmpty.ID, host.ID, nil, fetchedAt.Add(time.Minute)); err != nil {
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
	store, _, hostStore, ctx := newIntegrationReportStore(t)
	host := enrollTestHost(t, ctx, hostStore, "report-overwrite-host")
	report, err := store.Create(ctx, ReportMutation{
		Name:             "Overwrite report",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
	})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}

	firstFetchedAt := time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC)
	if err := store.OverwriteResults(ctx, report.ID, host.ID, []map[string]string{
		{"name": "Alpha"},
		{"name": "Bravo"},
	}, firstFetchedAt); err != nil {
		t.Fatalf("overwrite first snapshot: %v", err)
	}
	secondFetchedAt := firstFetchedAt.Add(time.Hour)
	if err := store.OverwriteResults(ctx, report.ID, host.ID, []map[string]string{
		{"name": "Charlie"},
	}, secondFetchedAt); err != nil {
		t.Fatalf("overwrite second snapshot: %v", err)
	}

	got, lastFetched, err := store.HostResults(ctx, host.ID, report.ID)
	if err != nil {
		t.Fatalf("host results: %v", err)
	}
	if len(got) != 1 || got[0].Columns["name"] != "Charlie" {
		t.Fatalf("HostResults = %+v, want only replacement row", got)
	}
	if lastFetched == nil || !lastFetched.Equal(secondFetchedAt) {
		t.Fatalf("last fetched = %v, want %s", lastFetched, secondFetchedAt)
	}

	emptyFetchedAt := secondFetchedAt.Add(time.Hour)
	if err := store.OverwriteResults(ctx, report.ID, host.ID, nil, emptyFetchedAt); err != nil {
		t.Fatalf("overwrite empty snapshot: %v", err)
	}
	got, lastFetched, err = store.HostResults(ctx, host.ID, report.ID)
	if err != nil {
		t.Fatalf("host results after empty snapshot: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("HostResults after empty snapshot = %+v, want no data rows", got)
	}
	if lastFetched == nil || !lastFetched.Equal(emptyFetchedAt) {
		t.Fatalf("empty last fetched = %v, want %s", lastFetched, emptyFetchedAt)
	}
}

func newIntegrationReportStore(t *testing.T) (*Store, *labels.Store, *hosts.Store, context.Context) {
	t.Helper()
	database, ctx := dbtest.Open(t)
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
	osqueryVersion string,
) *hosts.Host {
	t.Helper()
	host, err := store.UpsertOnOsqueryEnroll(ctx, hosts.InventoryUpdate{
		Hardware:       hosts.HostHardware{UUID: hardwareUUID},
		OsqueryNodeKey: hardwareUUID + "-node-key",
		Agents:         hosts.HostAgents{Osquery: hosts.HostOsqueryAgent{Version: osqueryVersion}},
	})
	if err != nil {
		t.Fatalf("enroll osquery host: %v", err)
	}
	return host
}

func allHostsLabelID(t *testing.T, ctx context.Context, store *labels.Store) int64 {
	t.Helper()
	rows, _, err := store.List(ctx, labels.ListParams{})
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
