package reports

import (
	"context"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func TestListIncludesLabelScope(t *testing.T) {
	store, labelStore, _, ctx := newIntegrationReportStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Report A")
	labelB := createManualLabel(t, ctx, labelStore, "Report B")

	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Scoped report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		LabelScope: scope.LabelScope{
			Mode:     scope.ScopeIncludeAll,
			LabelIDs: []int64{labelB.ID, labelA.ID, labelA.ID},
		},
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
	if got[0].LabelScope.Mode != scope.ScopeIncludeAll {
		t.Fatalf("LabelScope.Mode = %q, want %q", got[0].LabelScope.Mode, scope.ScopeIncludeAll)
	}
	assertInt64s(t, "LabelScope.LabelIDs", got[0].LabelScope.LabelIDs, []int64{labelA.ID, labelB.ID})
}

func TestScheduledForHostUsesLabelScope(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-scope-host", "5.22.1")
	matching := createManualLabel(t, ctx, labelStore, "Report match")
	other := createManualLabel(t, ctx, labelStore, "Report other")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}

	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Matching scheduled report",
		Query:            "select 1;",
		ScheduleInterval: 60,
		LabelScope:       scope.LabelScope{Mode: scope.ScopeIncludeAny, LabelIDs: []int64{matching.ID}},
	}); err != nil {
		t.Fatalf("create matching report: %v", err)
	}
	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Nonmatching scheduled report",
		Query:            "select 2;",
		ScheduleInterval: 60,
		LabelScope:       scope.LabelScope{Mode: scope.ScopeIncludeAll, LabelIDs: []int64{matching.ID, other.ID}},
	}); err != nil {
		t.Fatalf("create nonmatching report: %v", err)
	}

	got, err := store.ScheduledForHost(ctx, host)
	if err != nil {
		t.Fatalf("scheduled for host: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Matching scheduled report" {
		t.Fatalf("ScheduledForHost returned %+v, want only matching report", got)
	}
}

func TestScheduledForHostUsesScheduleState(t *testing.T) {
	store, _, hostStore, ctx := newIntegrationReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-applicable-host", "5.22.1")

	if _, err := store.Create(ctx, ReportMutation{
		Name:              "Matching scheduled report",
		Query:             "select 1;",
		MinOsqueryVersion: new("5.0.0"),
		ScheduleInterval:  60,
	}); err != nil {
		t.Fatalf("create matching report: %v", err)
	}
	if _, err := store.Create(ctx, ReportMutation{
		Name:             "Unscheduled report",
		Query:            "select 2;",
		ScheduleInterval: 0,
	}); err != nil {
		t.Fatalf("create unscheduled report: %v", err)
	}
	if _, err := store.Create(ctx, ReportMutation{
		Name:              "Version-gated scheduled report",
		Query:             "select 4;",
		MinOsqueryVersion: new("6.0.0"),
		ScheduleInterval:  60,
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
	store, _, hostStore, ctx := newIntegrationReportStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "report-host", "5.22.1")
	fetchedAt := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)

	reportWithRows, err := store.Create(ctx, ReportMutation{
		Name:             "Report with rows",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
	})
	if err != nil {
		t.Fatalf("create report with rows: %v", err)
	}
	reportEmpty, err := store.Create(ctx, ReportMutation{
		Name:             "Report empty",
		Query:            "select name from missing_apps;",
		ScheduleInterval: 60,
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

func assertInt64s(t *testing.T, name string, got []int64, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}
