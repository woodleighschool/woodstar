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

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/listing"
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
	if !errors.Is(err, fault.ErrNotFound) {
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

func TestHostSnapshotsIncludeCompleteLatestState(t *testing.T) {
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
	reportOff, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Report switched off",
		Query:            "select name from apps;",
		ScheduleInterval: 0,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}})
	if err != nil {
		t.Fatalf("create disabled report: %v", err)
	}
	if err := store.OverwriteSnapshot(ctx, reportWithRows.ID, testQueryHash(reportWithRows.Query), host.ID, []map[string]string{
		{"name": "Alpha"},
		{"name": "Bravo"},
	}, fetchedAt); err != nil {
		t.Fatalf("overwrite report rows: %v", err)
	}
	if err := store.OverwriteSnapshot(
		ctx,
		reportEmpty.ID,
		testQueryHash(reportEmpty.Query),
		host.ID,
		nil,
		fetchedAt.Add(time.Minute),
	); err != nil {
		t.Fatalf("overwrite empty report: %v", err)
	}

	got, _, err := store.HostSnapshots(ctx, host, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("host reports: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("HostSnapshots returned %d reports, want 3: %+v", len(got), got)
	}
	byID := make(map[int64]ReportSnapshot, len(got))
	for _, report := range got {
		byID[report.ReportID] = report
	}

	withRows := byID[reportWithRows.ID]
	if len(withRows.Rows) != 2 ||
		withRows.Rows[0]["name"] != "Alpha" ||
		withRows.Rows[1]["name"] != "Bravo" {
		t.Fatalf("Rows = %#v, want complete Alpha and Bravo snapshot", withRows.Rows)
	}
	if withRows.CollectedAt == nil || !withRows.CollectedAt.Equal(fetchedAt) {
		t.Fatalf("CollectedAt = %v, want %s", withRows.CollectedAt, fetchedAt)
	}

	empty := byID[reportEmpty.ID]
	if len(empty.Rows) != 0 {
		t.Fatalf("empty Rows = %#v, want empty snapshot", empty.Rows)
	}
	wantEmptyFetched := fetchedAt.Add(time.Minute)
	if empty.CollectedAt == nil || !empty.CollectedAt.Equal(wantEmptyFetched) {
		t.Fatalf("empty CollectedAt = %v, want %s", empty.CollectedAt, wantEmptyFetched)
	}

	off := byID[reportOff.ID]
	if off.CollectedAt != nil || off.Rows == nil || len(off.Rows) != 0 {
		t.Fatalf("disabled report snapshot = %+v, want pending assigned report", off)
	}
	if withRows.Status != ReportSnapshotStatusCollected ||
		empty.Status != ReportSnapshotStatusCollected ||
		off.Status != ReportSnapshotStatusPending {
		t.Fatalf(
			"host report statuses = rows %q empty %q off %q, want collected/collected/pending",
			withRows.Status,
			empty.Status,
			off.Status,
		)
	}

	pendingOnly, _, err := store.HostSnapshots(ctx, host, ReportSnapshotListParams{
		Status: ReportSnapshotStatusPending,
	})
	if err != nil {
		t.Fatalf("pending host reports: %v", err)
	}
	if len(pendingOnly) != 1 || pendingOnly[0].ReportID != reportOff.ID {
		t.Fatalf("pending host reports = %+v, want disabled report", pendingOnly)
	}
}

func TestSnapshotsIncludePendingTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	collectedHost := enrollTestHost(t, ctx, hostStore, "report-collected-host")
	pendingHost := enrollTestHost(t, ctx, hostStore, "report-pending-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	report, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Coverage report",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	collectedAt := time.Date(2026, 7, 29, 9, 0, 0, 0, time.UTC)
	if err := store.OverwriteSnapshot(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		collectedHost.ID,
		[]map[string]string{{"name": "Alpha"}},
		collectedAt,
	); err != nil {
		t.Fatalf("store collected snapshot: %v", err)
	}

	got, _, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("list report snapshots: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Snapshots returned %d hosts, want 2: %+v", len(got), got)
	}
	byHostID := make(map[int64]ReportSnapshot, len(got))
	for _, snapshot := range got {
		byHostID[snapshot.HostID] = snapshot
	}
	collected := byHostID[collectedHost.ID]
	if collected.Status != ReportSnapshotStatusCollected ||
		collected.CollectedAt == nil || !collected.CollectedAt.Equal(collectedAt) ||
		len(collected.Rows) != 1 || collected.Rows[0]["name"] != "Alpha" {
		t.Fatalf("collected snapshot = %+v, want Alpha observation", collected)
	}
	pending := byHostID[pendingHost.ID]
	if pending.Status != ReportSnapshotStatusPending ||
		pending.CollectedAt != nil || len(pending.Rows) != 0 {
		t.Fatalf("pending snapshot = %+v, want unfetched target", pending)
	}

	collectedOnly, _, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{
		Status: ReportSnapshotStatusCollected,
	})
	if err != nil {
		t.Fatalf("list collected report snapshots: %v", err)
	}
	if len(collectedOnly) != 1 || collectedOnly[0].HostID != collectedHost.ID {
		t.Fatalf("collected snapshots = %+v, want collected host", collectedOnly)
	}
	pendingOnly, _, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{
		Status: ReportSnapshotStatusPending,
	})
	if err != nil {
		t.Fatalf("list pending report snapshots: %v", err)
	}
	if len(pendingOnly) != 1 || pendingOnly[0].HostID != pendingHost.ID {
		t.Fatalf("pending snapshots = %+v, want pending host", pendingOnly)
	}
}

func TestSnapshotsSearchesParentAndPrunesNestedRows(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	matchingHost := enrollTestHost(t, ctx, hostStore, "report-search-matching-host")
	otherHost := enrollTestHost(t, ctx, hostStore, "report-search-other-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	report, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Searchable report",
		Query:            "select command from shell_history;",
		ScheduleInterval: 60,
		Targets:          reportTargets([]int64{allHostsID}, nil),
	}})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	collectedAt := time.Date(2026, 7, 29, 9, 0, 0, 0, time.UTC)
	if err := store.OverwriteSnapshot(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		matchingHost.ID,
		[]map[string]string{
			{"command": "sudo jamf manage"},
			{"command": "ping 10.10.0.1"},
		},
		collectedAt,
	); err != nil {
		t.Fatalf("store matching snapshot: %v", err)
	}
	if err := store.OverwriteSnapshot(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		otherHost.ID,
		[]map[string]string{{"command": "uptime"}},
		collectedAt,
	); err != nil {
		t.Fatalf("store other snapshot: %v", err)
	}

	nestedMatch, count, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{
		ListParams: listing.Params{Q: "sudo"},
	})
	if err != nil {
		t.Fatalf("search report rows: %v", err)
	}
	if count != 1 || len(nestedMatch) != 1 {
		t.Fatalf("nested search returned count=%d len=%d, want one snapshot", count, len(nestedMatch))
	}
	if len(nestedMatch[0].Rows) != 1 ||
		nestedMatch[0].Rows[0]["command"] != "sudo jamf manage" ||
		nestedMatch[0].ResultRowCount != 2 ||
		nestedMatch[0].ReturnedRowCount != 1 {
		t.Fatalf("nested search snapshot = %+v, want one of two matching rows", nestedMatch[0])
	}

	parentMatch, count, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{
		ListParams: listing.Params{Q: "matching-host"},
	})
	if err != nil {
		t.Fatalf("search report host: %v", err)
	}
	if count != 1 || len(parentMatch) != 1 ||
		parentMatch[0].HostID != matchingHost.ID ||
		len(parentMatch[0].Rows) != 2 ||
		parentMatch[0].ResultRowCount != 2 ||
		parentMatch[0].ReturnedRowCount != 2 {
		t.Fatalf("parent search snapshot = %+v count=%d, want complete matching-host snapshot", parentMatch, count)
	}

	page, count, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{
		ListParams: listing.Params{PageSize: 1, Sort: "host_name.desc"},
	})
	if err != nil {
		t.Fatalf("paginate report snapshots: %v", err)
	}
	if count != 2 || len(page) != 1 || page[0].HostID != otherHost.ID {
		t.Fatalf("paginated snapshots = %+v count=%d, want descending first of two", page, count)
	}
}

func TestSnapshotsRejectUnknownReport(t *testing.T) {
	store, _, _, ctx := newPostgresReportStore(t)

	_, _, err := store.Snapshots(ctx, 999_999, ReportSnapshotListParams{})
	if !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("Snapshots error = %v, want ErrNotFound", err)
	}
}

func TestOverwriteSnapshotReplacesHostStateAndRejectsOlderObservations(t *testing.T) {
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
	if err := store.OverwriteSnapshot(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Alpha"},
		{"name": "Bravo"},
	}, firstFetchedAt); err != nil {
		t.Fatalf("overwrite first snapshot: %v", err)
	}
	secondFetchedAt := firstFetchedAt.Add(time.Hour)
	if err := store.OverwriteSnapshot(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Charlie"},
	}, secondFetchedAt); err != nil {
		t.Fatalf("overwrite second snapshot: %v", err)
	}

	if err := store.OverwriteSnapshot(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		host.ID,
		[]map[string]string{{"name": "Outdated"}},
		firstFetchedAt.Add(30*time.Minute),
	); err != nil {
		t.Fatalf("overwrite stale snapshot: %v", err)
	}

	got, _, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("report snapshots: %v", err)
	}
	if len(got) != 1 || len(got[0].Rows) != 1 || got[0].Rows[0]["name"] != "Charlie" {
		t.Fatalf("Snapshots = %+v, want only newer Charlie snapshot", got)
	}

	emptyFetchedAt := secondFetchedAt.Add(time.Hour)
	if err := store.OverwriteSnapshot(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		host.ID,
		nil,
		emptyFetchedAt,
	); err != nil {
		t.Fatalf("overwrite empty snapshot: %v", err)
	}
	got, _, err = store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("report snapshots after empty observation: %v", err)
	}
	if len(got) != 1 || len(got[0].Rows) != 0 ||
		got[0].CollectedAt == nil || !got[0].CollectedAt.Equal(emptyFetchedAt) {
		t.Fatalf("Snapshots after empty observation = %+v, want collected empty snapshot", got)
	}
}

func TestUpdateInvalidatesResultsWhenQueryChanges(t *testing.T) {
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
	if err := store.OverwriteSnapshot(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
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
	got, _, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("results after metadata edit: %v", err)
	}
	if len(got) != 1 || len(got[0].Rows) != 1 {
		t.Fatalf("snapshots after metadata edit = %+v, want saved observation", got)
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
	got, _, err = store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("results after query edit: %v", err)
	}
	if len(got) != 1 || got[0].CollectedAt != nil || len(got[0].Rows) != 0 {
		t.Fatalf("snapshots after query edit = %+v, want pending target", got)
	}
	if err := store.OverwriteSnapshot(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		host.ID,
		[]map[string]string{{"name": "Obsolete"}},
		fetchedAt.Add(time.Minute),
	); err != nil {
		t.Fatalf("store obsolete report result: %v", err)
	}
	got, _, err = store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("results after obsolete snapshot: %v", err)
	}
	if len(got) != 1 || got[0].CollectedAt != nil || len(got[0].Rows) != 0 {
		t.Fatalf("snapshots after obsolete observation = %+v, want pending target", got)
	}
	if err := store.OverwriteSnapshot(
		ctx,
		report.ID,
		testQueryHash(queryUpdated.Query),
		host.ID,
		[]map[string]string{{"name": "Current"}},
		fetchedAt.Add(2*time.Minute),
	); err != nil {
		t.Fatalf("store current report result: %v", err)
	}
	got, _, err = store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("results after current snapshot: %v", err)
	}
	if len(got) != 1 || len(got[0].Rows) != 1 || got[0].Rows[0]["name"] != "Current" {
		t.Fatalf("snapshots after current observation = %+v, want Current", got)
	}
}

func TestUpdateInvalidatesResultsWhenMinimumVersionChanges(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresReportStore(t)
	host := enrollTestHost(t, ctx, hostStore, "report-version-change-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	targets := reportTargets([]int64{allHostsID}, nil)
	report, err := store.Create(ctx, ReportCreateMutation{ReportMutation: ReportMutation{
		Name:             "Version change report",
		Query:            "select name from apps;",
		ScheduleInterval: 60,
		Targets:          targets,
	}})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	fetchedAt := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	if err := store.OverwriteSnapshot(
		ctx,
		report.ID,
		testQueryHash(report.Query),
		host.ID,
		[]map[string]string{{"name": "Alpha"}},
		fetchedAt,
	); err != nil {
		t.Fatalf("store report result: %v", err)
	}

	versionUpdated, err := store.Update(ctx, report.ID, ReportMutation{
		Name:              report.Name,
		Query:             report.Query,
		MinOsqueryVersion: new("5.18.1"),
		ScheduleInterval:  report.ScheduleInterval,
		Targets:           targets,
	})
	if err != nil {
		t.Fatalf("update minimum osquery version: %v", err)
	}
	if versionUpdated.MinOsqueryVersion == nil || *versionUpdated.MinOsqueryVersion != "5.18.1" {
		t.Fatalf(
			"minimum osquery version = %v, want 5.18.1",
			versionUpdated.MinOsqueryVersion,
		)
	}
	got, _, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("results after minimum version edit: %v", err)
	}
	if len(got) != 1 || got[0].CollectedAt != nil || len(got[0].Rows) != 0 {
		t.Fatalf("snapshots after minimum version edit = %+v, want pending target", got)
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
	if err := store.OverwriteSnapshot(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Alpha"},
	}, fetchedAt); err != nil {
		t.Fatalf("store report result: %v", err)
	}

	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("exclude report host: %v", err)
	}
	assertReportSnapshotCount(t, ctx, store, report.ID, host.ID, 1)
	got, _, err := store.Snapshots(ctx, report.ID, ReportSnapshotListParams{})
	if err != nil {
		t.Fatalf("results after host exclusion: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("results after host exclusion = %+v, want none", got)
	}

	if err := store.OverwriteSnapshot(ctx, report.ID, testQueryHash(report.Query), host.ID, []map[string]string{
		{"name": "Late result"},
	}, fetchedAt.Add(time.Minute)); err != nil {
		t.Fatalf("store late unassigned result: %v", err)
	}
	assertReportSnapshotCount(t, ctx, store, report.ID, host.ID, 1)
	var persistedName string
	if err := store.pool.QueryRow(ctx, `
			SELECT rows->0->>'name'
			FROM osquery_report_snapshots
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
		if err := store.OverwriteSnapshot(ctx, report.ID, testQueryHash(report.Query), hostID, []map[string]string{
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

	rows, err := store.pool.Query(ctx, `
		SELECT DISTINCT host_id
			FROM osquery_report_snapshots
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

func assertReportSnapshotCount(
	t *testing.T,
	ctx context.Context,
	store *Store,
	reportID int64,
	hostID int64,
	want int,
) {
	t.Helper()
	var got int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
			FROM osquery_report_snapshots
		WHERE report_id = $1 AND host_id = $2`,
		reportID,
		hostID,
	).Scan(&got); err != nil {
		t.Fatalf("count persisted report snapshots: %v", err)
	}
	if got != want {
		t.Fatalf("persisted report snapshots = %d, want %d", got, want)
	}
}

func newPostgresReportStore(t *testing.T) (*Store, *labels.Store, *hosts.Store, context.Context) {
	t.Helper()
	database, ctx := testdb.Open(t)
	labelStore := labels.NewStore(database)
	return NewStore(database), labelStore, hosts.NewStore(database, labelStore), ctx
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
