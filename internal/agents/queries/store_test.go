package queries

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func TestCleanQueryCreate(t *testing.T) {
	tests := []struct {
		name    string
		in      QueryCreate
		want    QueryCreate
		wantErr string
	}{
		{
			name: "saved query trims fields",
			in: QueryCreate{
				Name:        " Local admins ",
				Description: " Users with admin rights ",
				Query:       " select * from users; ",
				Platform:    new(" darwin "),
			},
			want: QueryCreate{
				Name:        "Local admins",
				Description: "Users with admin rights",
				Query:       "select * from users;",
				Platform:    new("darwin"),
			},
		},
		{
			name: "scheduled report keeps interval",
			in: QueryCreate{
				Name:             "Battery health",
				Query:            "select * from battery;",
				ScheduleInterval: 3600,
			},
			want: QueryCreate{
				Name:             "Battery health",
				Query:            "select * from battery;",
				ScheduleInterval: 3600,
			},
		},
		{
			name:    "missing name is invalid",
			in:      QueryCreate{Query: "select 1;"},
			wantErr: "name is required",
		},
		{
			name:    "missing sql is invalid",
			in:      QueryCreate{Name: "No SQL"},
			wantErr: "query is required",
		},
		{
			name: "negative schedule is invalid",
			in: QueryCreate{
				Name:             "Bad schedule",
				Query:            "select 1;",
				ScheduleInterval: -1,
			},
			wantErr: "schedule interval cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := cleanQueryCreate(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("cleanQueryCreate error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("cleanQueryCreate returned error: %v", err)
			}
			assertQueryCreate(t, got, tt.want)
		})
	}
}

func TestListIncludesLabelScope(t *testing.T) {
	store, labelStore, _, ctx := newIntegrationQueryStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Query A")
	labelB := createManualLabel(t, ctx, labelStore, "Query B")

	if _, err := store.Create(ctx, QueryCreate{
		Name:             "Scoped query",
		Query:            "select 1;",
		ScheduleInterval: 60,
		LabelScope: scope.LabelScope{
			Mode:     scope.ScopeIncludeAll,
			LabelIDs: []int64{labelB.ID, labelA.ID, labelA.ID},
		},
	}); err != nil {
		t.Fatalf("create query: %v", err)
	}

	got, count, err := store.List(ctx, QueryListParams{})
	if err != nil {
		t.Fatalf("list queries: %v", err)
	}
	if count != 1 || len(got) != 1 {
		t.Fatalf("List returned count=%d len=%d, want one query", count, len(got))
	}
	if got[0].LabelScope.Mode != scope.ScopeIncludeAll {
		t.Fatalf("LabelScope.Mode = %q, want %q", got[0].LabelScope.Mode, scope.ScopeIncludeAll)
	}
	assertInt64s(t, "LabelScope.LabelIDs", got[0].LabelScope.LabelIDs, []int64{labelA.ID, labelB.ID})
}

func TestScheduledForHostUsesLabelScope(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationQueryStore(t)
	host := enrollTestHost(t, ctx, hostStore, "query-scope-host")
	matching := createManualLabel(t, ctx, labelStore, "Query match")
	other := createManualLabel(t, ctx, labelStore, "Query other")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}

	if _, err := store.Create(ctx, QueryCreate{
		Name:             "Matching scheduled query",
		Query:            "select 1;",
		ScheduleInterval: 60,
		LabelScope:       scope.LabelScope{Mode: scope.ScopeIncludeAny, LabelIDs: []int64{matching.ID}},
	}); err != nil {
		t.Fatalf("create matching query: %v", err)
	}
	if _, err := store.Create(ctx, QueryCreate{
		Name:             "Nonmatching scheduled query",
		Query:            "select 2;",
		ScheduleInterval: 60,
		LabelScope:       scope.LabelScope{Mode: scope.ScopeIncludeAll, LabelIDs: []int64{matching.ID, other.ID}},
	}); err != nil {
		t.Fatalf("create nonmatching query: %v", err)
	}

	got, err := store.ScheduledForHost(ctx, host)
	if err != nil {
		t.Fatalf("scheduled for host: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Matching scheduled query" {
		t.Fatalf("ScheduledForHost returned %+v, want only matching query", got)
	}
}

func TestScheduledForHostUsesHostApplicability(t *testing.T) {
	store, _, hostStore, ctx := newIntegrationQueryStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "query-applicable-host", "darwin", "5.22.1")

	if _, err := store.Create(ctx, QueryCreate{
		Name:              "Matching scheduled query",
		Query:             "select 1;",
		Platform:          new("darwin"),
		MinOsqueryVersion: new("5.0.0"),
		ScheduleInterval:  60,
	}); err != nil {
		t.Fatalf("create matching query: %v", err)
	}
	if _, err := store.Create(ctx, QueryCreate{
		Name:             "Unscheduled query",
		Query:            "select 2;",
		Platform:         new("darwin"),
		ScheduleInterval: 0,
	}); err != nil {
		t.Fatalf("create unscheduled query: %v", err)
	}
	if _, err := store.Create(ctx, QueryCreate{
		Name:             "Wrong platform query",
		Query:            "select 3;",
		Platform:         new("windows"),
		ScheduleInterval: 60,
	}); err != nil {
		t.Fatalf("create wrong platform query: %v", err)
	}
	if _, err := store.Create(ctx, QueryCreate{
		Name:              "Version-gated scheduled query",
		Query:             "select 4;",
		MinOsqueryVersion: new("6.0.0"),
		ScheduleInterval:  60,
	}); err != nil {
		t.Fatalf("create version-gated query: %v", err)
	}

	got, err := store.ScheduledForHost(ctx, host)
	if err != nil {
		t.Fatalf("scheduled for host: %v", err)
	}
	if len(got) != 2 || got[0].Name != "Matching scheduled query" || got[1].Name != "Version-gated scheduled query" {
		t.Fatalf("ScheduledForHost returned %+v, want matching platform/schedule queries", got)
	}
	if got[1].MinOsqueryVersion == nil || *got[1].MinOsqueryVersion != "6.0.0" {
		t.Fatalf("ScheduledForHost min version = %v, want preserved schedule metadata", got[1].MinOsqueryVersion)
	}
}

func TestHostReportsIncludeLatestHostState(t *testing.T) {
	store, _, hostStore, ctx := newIntegrationQueryStore(t)
	host := enrollTestHost(t, ctx, hostStore, "query-report-host")
	fetchedAt := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)

	reportWithRows, err := store.Create(ctx, QueryCreate{
		Name:             "Report with rows",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
	})
	if err != nil {
		t.Fatalf("create report with rows: %v", err)
	}
	reportEmpty, err := store.Create(ctx, QueryCreate{
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

func assertQueryCreate(t *testing.T, got QueryCreate, want QueryCreate) {
	t.Helper()
	if got.Name != want.Name {
		t.Fatalf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Fatalf("Description = %q, want %q", got.Description, want.Description)
	}
	if got.Query != want.Query {
		t.Fatalf("Query = %q, want %q", got.Query, want.Query)
	}
	if got.ScheduleInterval != want.ScheduleInterval {
		t.Fatalf("ScheduleInterval = %d, want %d", got.ScheduleInterval, want.ScheduleInterval)
	}
	assertStringPtr(t, "Platform", got.Platform, want.Platform)
}

func assertStringPtr(t *testing.T, name string, got *string, want *string) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("%s = %v, want %v", name, got, want)
	case *got != *want:
		t.Fatalf("%s = %q, want %q", name, *got, *want)
	}
}

func newIntegrationQueryStore(t *testing.T) (*Store, *labels.Store, *hosts.Store, context.Context) {
	t.Helper()
	database, ctx := dbtest.Open(t)
	return NewStore(database), labels.NewStore(database), hosts.NewStore(database), ctx
}

func createManualLabel(t *testing.T, ctx context.Context, store *labels.Store, name string) *labels.Label {
	t.Helper()
	label, err := store.Create(ctx, labels.LabelCreate{
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
	host, err := store.UpsertOnOrbitEnroll(ctx, hosts.EnrollParams{
		HardwareUUID: hardwareUUID,
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
	hostPlatform string,
	osqueryVersion string,
) *hosts.Host {
	t.Helper()
	host, err := store.UpsertOnOsqueryEnroll(ctx, hosts.HostDetailUpdate{
		HardwareUUID:   hardwareUUID,
		OsqueryNodeKey: hardwareUUID + "-node-key",
		Platform:       hostPlatform,
		OsqueryVersion: osqueryVersion,
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
