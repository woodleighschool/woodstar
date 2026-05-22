package checks

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func TestCleanCheckCreate(t *testing.T) {
	got, err := cleanCheckCreate(CheckCreate{
		Name:        " Gatekeeper enabled ",
		Description: " Security check ",
		Query:       " select 1 from gatekeeper where assessments_enabled = 1; ",
		Platforms:   []platforms.Platform{" darwin ", "DARWIN"},
	})
	if err != nil {
		t.Fatalf("cleanCheckCreate returned error: %v", err)
	}
	if got.Name != "Gatekeeper enabled" {
		t.Fatalf("Name = %q, want Gatekeeper enabled", got.Name)
	}
	if got.Query != "select 1 from gatekeeper where assessments_enabled = 1;" {
		t.Fatalf("Query = %q, want trimmed SQL", got.Query)
	}
	assertPlatforms(t, "Platforms", got.Platforms, []platforms.Platform{platforms.PlatformDarwin})
}

func TestListIncludesLabelScope(t *testing.T) {
	store, labelStore, _, ctx := newIntegrationCheckStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Check A")
	labelB := createManualLabel(t, ctx, labelStore, "Check B")

	if _, err := store.Create(ctx, CheckCreate{
		Name:      "Scoped check",
		Query:     "select 1;",
		Platforms: allPlatforms(),
		LabelScope: scope.LabelScope{
			Mode:     scope.ScopeExcludeAny,
			LabelIDs: []int64{labelB.ID, labelA.ID, labelA.ID},
		},
	}); err != nil {
		t.Fatalf("create check: %v", err)
	}

	got, count, err := store.List(ctx, CheckListParams{})
	if err != nil {
		t.Fatalf("list checks: %v", err)
	}
	if count != 1 || len(got) != 1 {
		t.Fatalf("List returned count=%d len=%d, want one check", count, len(got))
	}
	if got[0].LabelScope.Mode != scope.ScopeExcludeAny {
		t.Fatalf("LabelScope.Mode = %q, want %q", got[0].LabelScope.Mode, scope.ScopeExcludeAny)
	}
	assertInt64s(t, "LabelScope.LabelIDs", got[0].LabelScope.LabelIDs, []int64{labelA.ID, labelB.ID})
}

func TestListFiltersByPlatformTargetSet(t *testing.T) {
	store, _, _, ctx := newIntegrationCheckStore(t)
	if _, err := store.Create(ctx, CheckCreate{
		Name:      "All targets check",
		Query:     "select 1;",
		Platforms: allPlatforms(),
	}); err != nil {
		t.Fatalf("create all-target check: %v", err)
	}
	if _, err := store.Create(ctx, CheckCreate{
		Name:      "Windows only check",
		Query:     "select 2;",
		Platforms: []platforms.Platform{platforms.PlatformWindows},
	}); err != nil {
		t.Fatalf("create windows check: %v", err)
	}

	got, count, err := store.List(ctx, CheckListParams{Platform: "darwin"})
	if err != nil {
		t.Fatalf("list checks: %v", err)
	}
	if count != 1 || len(got) != 1 || got[0].Name != "All targets check" {
		t.Fatalf("List(platform=darwin) returned count=%d rows=%+v, want only all-target check", count, got)
	}
}

func TestApplicableForHostUsesLabelScope(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-scope-host", "darwin", "5.22.1")
	matching := createManualLabel(t, ctx, labelStore, "Check match")
	other := createManualLabel(t, ctx, labelStore, "Check other")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}

	if _, err := store.Create(ctx, CheckCreate{
		Name:       "Matching check",
		Query:      "select 1;",
		Platforms:  allPlatforms(),
		LabelScope: scope.LabelScope{Mode: scope.ScopeIncludeAny, LabelIDs: []int64{matching.ID}},
	}); err != nil {
		t.Fatalf("create matching check: %v", err)
	}
	if _, err := store.Create(ctx, CheckCreate{
		Name:       "Nonmatching check",
		Query:      "select 2;",
		Platforms:  allPlatforms(),
		LabelScope: scope.LabelScope{Mode: scope.ScopeIncludeAll, LabelIDs: []int64{matching.ID, other.ID}},
	}); err != nil {
		t.Fatalf("create nonmatching check: %v", err)
	}

	got, err := store.ApplicableForHost(ctx, host)
	if err != nil {
		t.Fatalf("applicable for host: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Matching check" {
		t.Fatalf("ApplicableForHost returned %+v, want only matching check", got)
	}
}

func TestHostChecksUseApplicability(t *testing.T) {
	store, _, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-applicable-host", "darwin", "5.22.1")

	matching, err := store.Create(ctx, CheckCreate{
		Name:      "Matching check",
		Query:     "select 1;",
		Platforms: []platforms.Platform{platforms.PlatformDarwin},
	})
	if err != nil {
		t.Fatalf("create matching check: %v", err)
	}
	wrongPlatform, err := store.Create(ctx, CheckCreate{
		Name:      "Wrong platform check",
		Query:     "select 2;",
		Platforms: []platforms.Platform{platforms.PlatformWindows},
	})
	if err != nil {
		t.Fatalf("create wrong platform check: %v", err)
	}
	for _, checkID := range []int64{matching.ID, wrongPlatform.ID} {
		passes := false
		if err := store.UpsertMembership(ctx, checkID, host.ID, &passes); err != nil {
			t.Fatalf("upsert membership for check %d: %v", checkID, err)
		}
	}

	got, err := store.HostChecks(ctx, host)
	if err != nil {
		t.Fatalf("host checks: %v", err)
	}
	if len(got) != 1 || got[0].CheckID != matching.ID {
		t.Fatalf("HostChecks returned %+v, want only matching check", got)
	}
}

func TestHostChecksIncludeMembershipState(t *testing.T) {
	store, _, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "check-status-host", "darwin", "5.22.1")

	passing, err := store.Create(ctx, CheckCreate{
		Name:      "Passing check",
		Query:     "select 1;",
		Platforms: allPlatforms(),
	})
	if err != nil {
		t.Fatalf("create passing check: %v", err)
	}
	failing, err := store.Create(ctx, CheckCreate{
		Name:      "Failing check",
		Query:     "select 0;",
		Platforms: allPlatforms(),
	})
	if err != nil {
		t.Fatalf("create failing check: %v", err)
	}
	unevaluated, err := store.Create(ctx, CheckCreate{
		Name:      "Unevaluated check",
		Query:     "select 2;",
		Platforms: allPlatforms(),
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
	store, _, hostStore, ctx := newIntegrationCheckStore(t)
	check, err := store.Create(ctx, CheckCreate{
		Name:      "Status list check",
		Query:     "select 1;",
		Platforms: allPlatforms(),
	})
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "aaa-failing-host", "darwin", "5.22.1")
	notRunHost := enrollTestHostDetail(t, ctx, hostStore, "bbb-not-run-host", "darwin", "5.22.1")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "ccc-passing-host", "darwin", "5.22.1")

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

func assertPlatforms(t *testing.T, name string, got []platforms.Platform, want []platforms.Platform) {
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

func allPlatforms() []platforms.Platform {
	return []platforms.Platform{platforms.PlatformDarwin, platforms.PlatformWindows, platforms.PlatformLinux}
}

func newIntegrationCheckStore(t *testing.T) (*Store, *labels.Store, *hosts.Store, context.Context) {
	t.Helper()
	database, ctx := dbtest.Open(t)
	return NewStore(database), labels.NewStore(database), hosts.NewStore(database), ctx
}

func createManualLabel(t *testing.T, ctx context.Context, store *labels.Store, name string) *labels.Label {
	t.Helper()
	label, err := store.Create(ctx, labels.LabelCreate{
		Name:                name,
		LabelMembershipType: labels.LabelMembershipTypeManual,
		Platforms:           allPlatforms(),
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
