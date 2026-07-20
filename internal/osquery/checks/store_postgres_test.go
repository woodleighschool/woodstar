//go:build postgres

package checks

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestListIncludesTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresCheckStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Check A")
	labelB := createManualLabel(t, ctx, labelStore, "Check B")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "check-list-passing-host")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "check-list-failing-host")

	check, err := store.Create(ctx, makeCheck(CheckMutation{
		Name:    "Targeted check",
		Query:   "select 1;",
		Targets: checkTargets([]int64{labelA.ID}, []int64{labelB.ID}),
	}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, check.ID, passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := store.UpsertMembership(ctx, check.ID, failingHost.ID, &fails); err != nil {
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
	if !errors.Is(err, dbutil.ErrNotFound) {
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
	if err := store.UpsertMembership(ctx, matching.ID, host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}

	got, err := store.HostChecks(ctx, host)
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
	if err := store.UpsertMembership(ctx, passing.ID, host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}
	fails := false
	if err := store.UpsertMembership(ctx, failing.ID, host.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}

	got, err := store.HostChecks(ctx, host)
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
	if passingStatus.Response == nil || *passingStatus.Response != CheckStatusPass {
		t.Fatalf("passing status Response = %v, want pass", passingStatus.Response)
	}
	if passingStatus.UpdatedAt == nil {
		t.Fatalf("passing status UpdatedAt is nil, want evaluated timestamp")
	}
	failingStatus := byID[failing.ID]
	if failingStatus.Response == nil || *failingStatus.Response != CheckStatusFail || failingStatus.UpdatedAt == nil {
		t.Fatalf("failing status = %+v, want fail with evaluated timestamp", failingStatus)
	}

	unevaluatedStatus := byID[unevaluated.ID]
	if unevaluatedStatus.Response != nil || unevaluatedStatus.UpdatedAt != nil {
		t.Fatalf("unevaluated status = %+v, want empty membership state", unevaluatedStatus)
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
	if err := store.UpsertMembership(ctx, check.ID, failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, check.ID, passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}

	got, err := store.CheckResults(ctx, check.ID, nil)
	if err != nil {
		t.Fatalf("check results: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("CheckResults returned %d hosts, want 3: %+v", len(got), got)
	}
	failStatus := CheckStatusFail
	passStatus := CheckStatusPass
	want := []struct {
		hostID   int64
		response *CheckStatus
		updated  bool
	}{
		{hostID: failingHost.ID, response: &failStatus, updated: true},
		{hostID: notRunHost.ID},
		{hostID: passingHost.ID, response: &passStatus, updated: true},
	}
	for i, wantStatus := range want {
		if got[i].HostID != wantStatus.hostID ||
			!equalCheckStatusPtr(got[i].Response, wantStatus.response) ||
			(got[i].UpdatedAt != nil) != wantStatus.updated {
			t.Fatalf(
				"CheckResults[%d] = %+v, want host=%d response=%v updated=%v",
				i,
				got[i],
				wantStatus.hostID,
				wantStatus.response,
				wantStatus.updated,
			)
		}
	}
}

func TestCheckResultsFiltersByMembershipStatus(t *testing.T) {
	store, _, hostStore, ctx := newPostgresCheckStore(t)
	check, err := store.Create(ctx, makeCheck(CheckMutation{Name: "Host ID status check", Query: "select 1"}))
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "check-host-id-passing")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "check-host-id-failing")
	unevaluatedHost := enrollTestHostDetail(t, ctx, hostStore, "check-host-id-unevaluated")

	passes := true
	if err := store.UpsertMembership(ctx, check.ID, passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := store.UpsertMembership(ctx, check.ID, failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	if err := store.UpsertMembership(ctx, check.ID, unevaluatedHost.ID, nil); err != nil {
		t.Fatalf("upsert unevaluated membership: %v", err)
	}

	passStatus := CheckStatusPass
	passingResults, err := store.CheckResults(ctx, check.ID, &passStatus)
	if err != nil {
		t.Fatalf("pass results: %v", err)
	}
	if len(passingResults) != 1 ||
		passingResults[0].HostID != passingHost.ID ||
		passingResults[0].HostName == "" ||
		passingResults[0].Response == nil ||
		*passingResults[0].Response != CheckStatusPass {
		t.Fatalf("pass results = %+v, want passing host status", passingResults)
	}

	failStatus := CheckStatusFail
	failingResults, err := store.CheckResults(ctx, check.ID, &failStatus)
	if err != nil {
		t.Fatalf("fail results: %v", err)
	}
	if len(failingResults) != 1 ||
		failingResults[0].HostID != failingHost.ID ||
		failingResults[0].HostName == "" ||
		failingResults[0].Response == nil ||
		*failingResults[0].Response != CheckStatusFail {
		t.Fatalf("fail results = %+v, want failing host status", failingResults)
	}
}

func equalCheckStatusPtr(a *CheckStatus, b *CheckStatus) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
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
