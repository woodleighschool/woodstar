package checks

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func TestListIncludesTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationCheckStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Check A")
	labelB := createManualLabel(t, ctx, labelStore, "Check B")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "check-list-passing-host", "5.22.1")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "check-list-failing-host", "5.22.1")

	check, err := store.Create(ctx, CheckMutation{
		Name:  "Targeted check",
		Query: "select 1;",
		Targets: []scope.TargetLabel{
			{LabelID: labelA.ID, Effect: scope.TargetLabelInclude},
			{LabelID: labelB.ID, Effect: scope.TargetLabelExclude},
		},
	})
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
	assertTargets(t, got[0].Targets, []scope.TargetLabel{
		{LabelID: labelB.ID, Effect: scope.TargetLabelExclude},
		{LabelID: labelA.ID, Effect: scope.TargetLabelInclude},
	})
}

func TestApplicableForHostUsesTargetRows(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-scope-host", "5.22.1")
	matching := createManualLabel(t, ctx, labelStore, "Check match")
	other := createManualLabel(t, ctx, labelStore, "Check other")
	excluded := createManualLabel(t, ctx, labelStore, "Check excluded")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, CheckMutation{
		Name:  "Matching check",
		Query: "select 1;",
		Targets: []scope.TargetLabel{
			{LabelID: matching.ID, Effect: scope.TargetLabelInclude},
		},
	}); err != nil {
		t.Fatalf("create matching check: %v", err)
	}
	if _, err := store.Create(ctx, CheckMutation{
		Name:  "Nonmatching check",
		Query: "select 2;",
		Targets: []scope.TargetLabel{
			{LabelID: other.ID, Effect: scope.TargetLabelInclude},
		},
	}); err != nil {
		t.Fatalf("create nonmatching check: %v", err)
	}
	if _, err := store.Create(ctx, CheckMutation{
		Name:  "Excluded check",
		Query: "select 3;",
		Targets: []scope.TargetLabel{
			{LabelID: matching.ID, Effect: scope.TargetLabelInclude},
			{LabelID: excluded.ID, Effect: scope.TargetLabelExclude},
		},
	}); err != nil {
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
	store, labelStore, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-requires-include-host", "5.22.1")
	excluded := createManualLabel(t, ctx, labelStore, "Check requires include excluded")
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, CheckMutation{
		Name:  "Exclude-only check",
		Query: "select 1;",
		Targets: []scope.TargetLabel{
			{LabelID: excluded.ID, Effect: scope.TargetLabelExclude},
		},
	}); err != nil {
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

func TestHostChecksIncludesMatchingChecks(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-applicable-host", "5.22.1")
	allHostsID := allHostsLabelID(t, ctx, labelStore)

	matching, err := store.Create(ctx, CheckMutation{
		Name:    "Matching check",
		Query:   "select 1;",
		Targets: []scope.TargetLabel{{LabelID: allHostsID, Effect: scope.TargetLabelInclude}},
	})
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
	store, labelStore, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-status-host", "5.22.1")
	allHostsID := allHostsLabelID(t, ctx, labelStore)

	passing, err := store.Create(ctx, CheckMutation{
		Name:    "Passing check",
		Query:   "select 1;",
		Targets: []scope.TargetLabel{{LabelID: allHostsID, Effect: scope.TargetLabelInclude}},
	})
	if err != nil {
		t.Fatalf("create passing check: %v", err)
	}
	failing, err := store.Create(ctx, CheckMutation{
		Name:    "Failing check",
		Query:   "select 0;",
		Targets: []scope.TargetLabel{{LabelID: allHostsID, Effect: scope.TargetLabelInclude}},
	})
	if err != nil {
		t.Fatalf("create failing check: %v", err)
	}
	unevaluated, err := store.Create(ctx, CheckMutation{
		Name:    "Unevaluated check",
		Query:   "select 2;",
		Targets: []scope.TargetLabel{{LabelID: allHostsID, Effect: scope.TargetLabelInclude}},
	})
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

func TestHostStatusesIncludeMembershipState(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationCheckStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	check, err := store.Create(ctx, CheckMutation{
		Name:    "Status list check",
		Query:   "select 1;",
		Targets: []scope.TargetLabel{{LabelID: allHostsID, Effect: scope.TargetLabelInclude}},
	})
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "aaa-failing-host", "5.22.1")
	notRunHost := enrollTestHostDetail(t, ctx, hostStore, "bbb-not-run-host", "5.22.1")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "ccc-passing-host", "5.22.1")

	fails := false
	if err := store.UpsertMembership(ctx, check.ID, failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, check.ID, passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}

	got, err := store.HostStatuses(ctx, check.ID)
	if err != nil {
		t.Fatalf("host statuses: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("HostStatuses returned %d hosts, want 3: %+v", len(got), got)
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
				"HostStatuses[%d] = %+v, want host=%d response=%v updated=%v",
				i,
				got[i],
				wantStatus.hostID,
				wantStatus.response,
				wantStatus.updated,
			)
		}
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

func newIntegrationCheckStore(t *testing.T) (*Store, *labels.Store, *hosts.Store, context.Context) {
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
		if row.Name == "All Hosts" {
			return row.ID
		}
	}
	t.Fatalf("All Hosts label not found")
	return 0
}

func assertTargets(t *testing.T, got []scope.TargetLabel, want []scope.TargetLabel) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("targets = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("targets = %#v, want %#v", got, want)
		}
	}
}
