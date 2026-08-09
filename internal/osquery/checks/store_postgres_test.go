//go:build postgres

package checks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestListIncludesTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Check A")
	labelB := createManualLabel(t, ctx, labelStore, "Check B")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "check-list-passing-host")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "check-list-failing-host")
	if err := labelStore.SetMembership(ctx, labelA.ID, passingHost.ID, true); err != nil {
		t.Fatalf("include passing host: %v", err)
	}
	if err := labelStore.SetMembership(ctx, labelA.ID, failingHost.ID, true); err != nil {
		t.Fatalf("include failing host: %v", err)
	}

	check, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Targeted check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{labelA.ID}, []int64{labelB.ID}),
	}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}

	got, count, err := store.List(ctx, CheckListParams{})
	if err != nil {
		t.Fatalf("list checks: %v", err)
	}
	if count != 1 || len(got) != 1 {
		t.Fatalf("List returned count=%d len=%d, want one check", count, len(got))
	}
	if got[0].PassingHostCount != 1 || got[0].FailingHostCount != 1 {
		t.Fatalf("host counts = pass %d fail %d, want 1/1", got[0].PassingHostCount, got[0].FailingHostCount)
	}
	assertTargets(t, got[0].Targets, checkTargets([]int64{labelA.ID}, []int64{labelB.ID}))
}

func TestCountsAndResultsUseCurrentTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	excluded := createManualLabel(t, ctx, labelStore, "Check excluded")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "check-current-passing-host")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "check-current-failing-host")
	excludedHost := enrollTestHostDetail(t, ctx, hostStore, "check-excluded-failing-host")

	check, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Current target check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{allHostsID}, []int64{excluded.ID}),
	}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), excludedHost.ID, &fails); err != nil {
		t.Fatalf("upsert excluded membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excluded.ID, excludedHost.ID, true); err != nil {
		t.Fatalf("exclude host: %v", err)
	}

	var persistedExcluded int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM osquery_check_membership
		WHERE check_id = $1 AND host_id = $2`,
		check.ID,
		excludedHost.ID,
	).Scan(&persistedExcluded); err != nil {
		t.Fatalf("count excluded membership: %v", err)
	}
	if persistedExcluded != 1 {
		t.Fatalf("excluded membership rows = %d, want retained historical row", persistedExcluded)
	}
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), excludedHost.ID, &passes); err != nil {
		t.Fatalf("upsert late excluded membership: %v", err)
	}
	var persistedPasses bool
	if err := store.pool.QueryRow(ctx, `
			SELECT passes
			FROM osquery_check_membership
			WHERE check_id = $1 AND host_id = $2`,
		check.ID,
		excludedHost.ID,
	).Scan(&persistedPasses); err != nil {
		t.Fatalf("read excluded membership: %v", err)
	}
	if persistedPasses {
		t.Fatal("late excluded result replaced the retained historical result")
	}

	got, err := store.GetByID(ctx, check.ID)
	if err != nil {
		t.Fatalf("get check: %v", err)
	}
	if got.PassingHostCount != 1 || got.FailingHostCount != 1 {
		t.Fatalf(
			"GetByID host counts = pass %d fail %d, want 1/1",
			got.PassingHostCount,
			got.FailingHostCount,
		)
	}

	listed, _, err := store.List(ctx, CheckListParams{})
	if err != nil {
		t.Fatalf("list checks: %v", err)
	}
	if len(listed) != 1 ||
		listed[0].PassingHostCount != 1 ||
		listed[0].FailingHostCount != 1 {
		t.Fatalf("List check = %+v, want pass 1 fail 1", listed)
	}

	results, _, err := store.CheckResults(ctx, check.ID, CheckResultListParams{
		Statuses: []CheckStatus{CheckStatusFail},
	})
	if err != nil {
		t.Fatalf("list failing results: %v", err)
	}
	if len(results) != 1 || results[0].HostID != failingHost.ID {
		t.Fatalf("failing results = %+v, want only current failing host", results)
	}
}

func TestUpdateInvalidatesMembershipOnlyWhenQueryChanges(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-query-change-host")
	targets := checkTargets([]int64{allHostsID}, nil)
	check, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Query change check",
		Query:   "select 1;",
		Targets: targets,
	}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}

	metadataUpdated, err := store.Update(ctx, check.ID, CheckMutation{
		Name:        "Renamed check",
		Description: "Non-semantic edit",
		Query:       " select 1; ",
		Targets:     targets,
	})
	if err != nil {
		t.Fatalf("update check metadata: %v", err)
	}
	if testQueryHash(metadataUpdated.Query) != testQueryHash(check.Query) {
		t.Fatal("metadata edit changed the normalized query hash")
	}
	got, err := store.GetByID(ctx, check.ID)
	if err != nil {
		t.Fatalf("get metadata-updated check: %v", err)
	}
	if got.PassingHostCount != 1 {
		t.Fatalf("passing count after metadata edit = %d, want 1", got.PassingHostCount)
	}

	queryUpdated, err := store.Update(ctx, check.ID, CheckMutation{
		Name:        "Renamed check",
		Description: "Non-semantic edit",
		Query:       "select 2;",
		Targets:     targets,
	})
	if err != nil {
		t.Fatalf("update check query: %v", err)
	}
	if testQueryHash(queryUpdated.Query) == testQueryHash(check.Query) {
		t.Fatal("changed query retained its previous query hash")
	}
	got, err = store.GetByID(ctx, check.ID)
	if err != nil {
		t.Fatalf("get query-updated check: %v", err)
	}
	if got.PassingHostCount != 0 || got.FailingHostCount != 0 {
		t.Fatalf(
			"host counts after query edit = pass %d fail %d, want 0/0",
			got.PassingHostCount,
			got.FailingHostCount,
		)
	}
	if err := store.UpsertMembership(
		ctx,
		check.ID,
		testQueryHash(check.Query),
		host.ID,
		&passes,
	); err != nil {
		t.Fatalf("upsert obsolete query result: %v", err)
	}
	got, err = store.GetByID(ctx, check.ID)
	if err != nil {
		t.Fatalf("get check after obsolete result: %v", err)
	}
	if got.PassingHostCount != 0 {
		t.Fatalf("passing count after obsolete result = %d, want 0", got.PassingHostCount)
	}
	if err := store.UpsertMembership(
		ctx,
		check.ID,
		testQueryHash(queryUpdated.Query),
		host.ID,
		&passes,
	); err != nil {
		t.Fatalf("upsert current query result: %v", err)
	}
	got, err = store.GetByID(ctx, check.ID)
	if err != nil {
		t.Fatalf("get check after current result: %v", err)
	}
	if got.PassingHostCount != 1 {
		t.Fatalf("passing count after current result = %d, want 1", got.PassingHostCount)
	}
}

func TestUpdatePrunesMembershipOutsideNewTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	retainedLabel := createManualLabel(t, ctx, labelStore, "Check retained")
	retainedHost := enrollTestHostDetail(t, ctx, hostStore, "check-retained-host")
	removedHost := enrollTestHostDetail(t, ctx, hostStore, "check-removed-host")
	if err := labelStore.SetMembership(ctx, retainedLabel.ID, retainedHost.ID, true); err != nil {
		t.Fatalf("retain host: %v", err)
	}

	check, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Retargeted check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	passes := true
	for _, hostID := range []int64{retainedHost.ID, removedHost.ID} {
		if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), hostID, &passes); err != nil {
			t.Fatalf("upsert host %d membership: %v", hostID, err)
		}
	}

	if _, err := store.Update(ctx, check.ID, CheckMutation{
		Name:    "Retargeted check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{retainedLabel.ID}, nil),
	}); err != nil {
		t.Fatalf("retarget check: %v", err)
	}

	var hostIDs []int64
	rows, err := store.pool.Query(ctx, `
		SELECT host_id
		FROM osquery_check_membership
		WHERE check_id = $1
		ORDER BY host_id`,
		check.ID,
	)
	if err != nil {
		t.Fatalf("list persisted memberships: %v", err)
	}
	hostIDs, err = pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		t.Fatalf("collect persisted memberships: %v", err)
	}
	if len(hostIDs) != 1 || hostIDs[0] != retainedHost.ID {
		t.Fatalf("persisted host IDs = %v, want [%d]", hostIDs, retainedHost.ID)
	}
}

func TestUpdateReplacesTargets(t *testing.T) {
	store, labelStore, _, ctx := newPostgresCheckStore(t)
	first := createManualLabel(t, ctx, labelStore, "Check first")
	second := createManualLabel(t, ctx, labelStore, "Check second")
	third := createManualLabel(t, ctx, labelStore, "Check third")

	check, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Replacement check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{first.ID, second.ID}, []int64{third.ID}),
	}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}

	updated, err := store.Update(ctx, check.ID, CheckMutation{
		Name:    "Replacement check",
		Query:   "select 2;",
		Targets: checkTargets([]int64{third.ID}, []int64{first.ID}),
	})
	if err != nil {
		t.Fatalf("update check: %v", err)
	}
	assertTargets(t, updated.Targets, checkTargets([]int64{third.ID}, []int64{first.ID}))

	got, err := store.GetByID(ctx, check.ID)
	if err != nil {
		t.Fatalf("get updated check: %v", err)
	}
	assertTargets(t, got.Targets, checkTargets([]int64{third.ID}, []int64{first.ID}))
}

func TestApplicableForHostUsesTargetRows(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-target-host")
	matching := createManualLabel(t, ctx, labelStore, "Check match")
	other := createManualLabel(t, ctx, labelStore, "Check other")
	excluded := createManualLabel(t, ctx, labelStore, "Check excluded")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Matching check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{matching.ID}, nil),
	})); err != nil {
		t.Fatalf("create matching check: %v", err)
	}
	if _, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Nonmatching check",
		Query:   "select 2;",
		Targets: checkTargets([]int64{other.ID}, nil),
	})); err != nil {
		t.Fatalf("create nonmatching check: %v", err)
	}
	if _, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Excluded check",
		Query:   "select 3;",
		Targets: checkTargets([]int64{matching.ID}, []int64{excluded.ID}),
	})); err != nil {
		t.Fatalf("create excluded check: %v", err)
	}

	got, err := store.ApplicableForHost(ctx, host)
	if err != nil {
		t.Fatalf("applicable for host: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Matching check" {
		t.Fatalf("ApplicableForHost returned %+v, want only matching check", got)
	}
}

func TestApplicableForHostRequiresIncludeTarget(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-requires-include-host")
	excluded := createManualLabel(t, ctx, labelStore, "Check requires include excluded")
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Exclude-only check",
		Query:   "select 1;",
		Targets: checkTargets(nil, []int64{excluded.ID}),
	})); err != nil {
		t.Fatalf("create exclude-only check: %v", err)
	}

	got, err := store.ApplicableForHost(ctx, host)
	if err != nil {
		t.Fatalf("applicable for host: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ApplicableForHost returned %+v, want no checks", got)
	}
}

func TestCreateCheckWithMissingLabelReturnsNotFound(t *testing.T) {
	store, _, _, ctx := newPostgresCheckStore(t)

	_, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Missing label target",
		Query:   "select 1;",
		Targets: checkTargets([]int64{999_999}, nil),
	}))
	if !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("Create error = %v, want ErrNotFound", err)
	}
}

func TestHostChecksIncludesMatchingChecks(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-applicable-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)

	matching, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Matching check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create matching check: %v", err)
	}
	passes := false
	if err := store.UpsertMembership(ctx, matching.ID, testQueryHash(matching.Query), host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}

	got, _, err := store.HostChecks(ctx, host, CheckResultListParams{})
	if err != nil {
		t.Fatalf("host checks: %v", err)
	}
	if len(got) != 1 || got[0].CheckID != matching.ID {
		t.Fatalf("HostChecks returned %+v, want matching check", got)
	}
}

func TestHostChecksIncludeMembershipState(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-status-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)

	passing, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Passing check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create passing check: %v", err)
	}
	failing, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Failing check",
		Query:   "select 0;",
		Targets: checkTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create failing check: %v", err)
	}
	unevaluated, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Unevaluated check",
		Query:   "select 2;",
		Targets: checkTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create unevaluated check: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, passing.ID, testQueryHash(passing.Query), host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}
	fails := false
	if err := store.UpsertMembership(ctx, failing.ID, testQueryHash(failing.Query), host.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}

	got, _, err := store.HostChecks(ctx, host, CheckResultListParams{})
	if err != nil {
		t.Fatalf("host checks: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("HostChecks returned %d checks, want 3: %+v", len(got), got)
	}
	wantOrder := []int64{failing.ID, unevaluated.ID, passing.ID}
	for i, wantID := range wantOrder {
		if got[i].CheckID != wantID {
			t.Fatalf("HostChecks order = %+v, want fail/not-run/pass", got)
		}
	}
	byID := make(map[int64]CheckHostStatus, len(got))
	for _, status := range got {
		byID[status.CheckID] = status
	}

	passingStatus := byID[passing.ID]
	if passingStatus.Status != CheckStatusPass {
		t.Fatalf("passing status = %q, want pass", passingStatus.Status)
	}
	if passingStatus.UpdatedAt == nil {
		t.Fatalf("passing status UpdatedAt is nil, want evaluated timestamp")
	}
	failingStatus := byID[failing.ID]
	if failingStatus.Status != CheckStatusFail || failingStatus.UpdatedAt == nil {
		t.Fatalf("failing status = %+v, want fail with evaluated timestamp", failingStatus)
	}

	unevaluatedStatus := byID[unevaluated.ID]
	if unevaluatedStatus.Status != CheckStatusPending || unevaluatedStatus.UpdatedAt != nil {
		t.Fatalf("unevaluated status = %+v, want pending without evaluation time", unevaluatedStatus)
	}
}

func TestCheckResultsIncludeMembershipState(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	check, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Status list check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "aaa-failing-host")
	notRunHost := enrollTestHostDetail(t, ctx, hostStore, "bbb-not-run-host")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "ccc-passing-host")

	fails := false
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}

	got, _, err := store.CheckResults(ctx, check.ID, CheckResultListParams{})
	if err != nil {
		t.Fatalf("check results: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("CheckResults returned %d hosts, want 3: %+v", len(got), got)
	}
	want := []struct {
		hostID  int64
		status  CheckStatus
		updated bool
	}{
		{hostID: failingHost.ID, status: CheckStatusFail, updated: true},
		{hostID: notRunHost.ID, status: CheckStatusPending},
		{hostID: passingHost.ID, status: CheckStatusPass, updated: true},
	}
	for i, wantStatus := range want {
		if got[i].HostID != wantStatus.hostID ||
			got[i].Status != wantStatus.status ||
			(got[i].UpdatedAt != nil) != wantStatus.updated {
			t.Fatalf(
				"CheckResults[%d] = %+v, want host=%d status=%v updated=%v",
				i,
				got[i],
				wantStatus.hostID,
				wantStatus.status,
				wantStatus.updated,
			)
		}
	}
}

func TestCheckResultsAndHostChecksSearchAndPaginate(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	firstCheck, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Alpha searchable check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create first check: %v", err)
	}
	if _, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Bravo other check",
		Query:   "select 2;",
		Targets: checkTargets([]int64{allHostsID}, nil),
	})); err != nil {
		t.Fatalf("create second check: %v", err)
	}
	matchingHost := enrollTestHostDetail(t, ctx, hostStore, "check-search-matching-host")
	otherHost := enrollTestHostDetail(t, ctx, hostStore, "check-search-other-host")

	hostResults, count, err := store.CheckResults(ctx, firstCheck.ID, CheckResultListParams{
		ListParams: listing.Params{
			Q:        "matching-host",
			PageSize: 1,
			Sort:     "host_name.desc",
		},
	})
	if err != nil {
		t.Fatalf("search check results: %v", err)
	}
	if count != 1 || len(hostResults) != 1 || hostResults[0].HostID != matchingHost.ID {
		t.Fatalf("check result search = %+v count=%d, want matching host", hostResults, count)
	}

	hostResults, count, err = store.CheckResults(ctx, firstCheck.ID, CheckResultListParams{
		ListParams: listing.Params{PageSize: 1, Sort: "host_name.desc"},
	})
	if err != nil {
		t.Fatalf("paginate check results: %v", err)
	}
	if count != 2 || len(hostResults) != 1 || hostResults[0].HostID != otherHost.ID {
		t.Fatalf("check result page = %+v count=%d, want descending first of two", hostResults, count)
	}

	checkResults, count, err := store.HostChecks(ctx, matchingHost, CheckResultListParams{
		ListParams: listing.Params{Q: "Alpha"},
	})
	if err != nil {
		t.Fatalf("search host checks: %v", err)
	}
	if count != 1 || len(checkResults) != 1 || checkResults[0].CheckID != firstCheck.ID {
		t.Fatalf("host check search = %+v count=%d, want Alpha check", checkResults, count)
	}
}

func TestCheckResultsFiltersByMembershipStatus(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	check, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Host ID status check",
		Query:   "select 1",
		Targets: checkTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "check-host-id-passing")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "check-host-id-failing")
	unevaluatedHost := enrollTestHostDetail(t, ctx, hostStore, "check-host-id-unevaluated")

	passes := true
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	if err := store.UpsertMembership(ctx, check.ID, testQueryHash(check.Query), unevaluatedHost.ID, nil); err != nil {
		t.Fatalf("upsert unevaluated membership: %v", err)
	}

	passingResults, _, err := store.CheckResults(ctx, check.ID, CheckResultListParams{
		Statuses: []CheckStatus{CheckStatusPass},
	})
	if err != nil {
		t.Fatalf("pass results: %v", err)
	}
	if len(passingResults) != 1 ||
		passingResults[0].HostID != passingHost.ID ||
		passingResults[0].HostName == "" ||
		passingResults[0].Status != CheckStatusPass {
		t.Fatalf("pass results = %+v, want passing host status", passingResults)
	}

	failingResults, _, err := store.CheckResults(ctx, check.ID, CheckResultListParams{
		Statuses: []CheckStatus{CheckStatusFail},
	})
	if err != nil {
		t.Fatalf("fail results: %v", err)
	}
	if len(failingResults) != 1 ||
		failingResults[0].HostID != failingHost.ID ||
		failingResults[0].HostName == "" ||
		failingResults[0].Status != CheckStatusFail {
		t.Fatalf("fail results = %+v, want failing host status", failingResults)
	}

	pendingResults, _, err := store.CheckResults(ctx, check.ID, CheckResultListParams{
		Statuses: []CheckStatus{CheckStatusPending},
	})
	if err != nil {
		t.Fatalf("pending results: %v", err)
	}
	if len(pendingResults) != 1 ||
		pendingResults[0].HostID != unevaluatedHost.ID ||
		pendingResults[0].Status != CheckStatusPending {
		t.Fatalf("pending results = %+v, want unevaluated host status", pendingResults)
	}

	completedResults, _, err := store.CheckResults(ctx, check.ID, CheckResultListParams{
		Statuses: []CheckStatus{CheckStatusPass, CheckStatusFail},
	})
	if err != nil {
		t.Fatalf("completed results: %v", err)
	}
	if len(completedResults) != 2 {
		t.Fatalf("completed results = %+v, want passing and failing hosts", completedResults)
	}
}

func newPostgresCheckStore(t *testing.T) (*Store, *labels.Store, *hosts.Store, context.Context) {
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

func makeCheck(m CheckMutation) CheckCreateMutation {
	return CheckCreateMutation{CheckMutation: m}
}

func checkTargets(includeIDs, excludeIDs []int64) CheckTargets {
	return CheckTargets{
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

func assertTargets(t *testing.T, got CheckTargets, want CheckTargets) {
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
