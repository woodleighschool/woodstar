package checks

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func TestCleanCheckCreate(t *testing.T) {
	got, err := cleanCheckCreate(CheckCreate{
		Name:        " Gatekeeper enabled ",
		Description: " Security check ",
		Query:       " select 1 from gatekeeper where assessments_enabled = 1; ",
		Platform:    new(" darwin "),
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
	assertStringPtr(t, "Platform", got.Platform, new("darwin"))
}

func TestListIncludesLabelScope(t *testing.T) {
	store, labelStore, _, ctx := newIntegrationCheckStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Check A")
	labelB := createManualLabel(t, ctx, labelStore, "Check B")

	if _, err := store.Create(ctx, CheckCreate{
		Name:  "Scoped check",
		Query: "select 1;",
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

func TestApplicableForHostUsesLabelScope(t *testing.T) {
	store, labelStore, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHost(t, ctx, hostStore, "check-scope-host")
	matching := createManualLabel(t, ctx, labelStore, "Check match")
	other := createManualLabel(t, ctx, labelStore, "Check other")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}

	if _, err := store.Create(ctx, CheckCreate{
		Name:       "Matching check",
		Query:      "select 1;",
		LabelScope: scope.LabelScope{Mode: scope.ScopeIncludeAny, LabelIDs: []int64{matching.ID}},
	}); err != nil {
		t.Fatalf("create matching check: %v", err)
	}
	if _, err := store.Create(ctx, CheckCreate{
		Name:       "Nonmatching check",
		Query:      "select 2;",
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

func TestHostChecksIncludeMembershipState(t *testing.T) {
	store, _, hostStore, ctx := newIntegrationCheckStore(t)
	host := enrollTestHost(t, ctx, hostStore, "check-status-host")

	passing, err := store.Create(ctx, CheckCreate{
		Name:  "Passing check",
		Query: "select 1;",
	})
	if err != nil {
		t.Fatalf("create passing check: %v", err)
	}
	unevaluated, err := store.Create(ctx, CheckCreate{
		Name:  "Unevaluated check",
		Query: "select 2;",
	})
	if err != nil {
		t.Fatalf("create unevaluated check: %v", err)
	}
	passes := true
	if err := store.UpsertMembership(ctx, passing.ID, host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}

	got, err := store.HostChecks(ctx, host)
	if err != nil {
		t.Fatalf("host checks: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("HostChecks returned %d checks, want 2: %+v", len(got), got)
	}
	byID := make(map[int64]CheckHostStatus, len(got))
	for _, status := range got {
		byID[status.CheckID] = status
	}

	passingStatus := byID[passing.ID]
	if passingStatus.Passes == nil || !*passingStatus.Passes {
		t.Fatalf("passing status Passes = %v, want true", passingStatus.Passes)
	}
	if passingStatus.LastEvaluatedAt == nil {
		t.Fatalf("passing status LastEvaluatedAt is nil, want evaluated timestamp")
	}

	unevaluatedStatus := byID[unevaluated.ID]
	if unevaluatedStatus.Passes != nil ||
		unevaluatedStatus.FirstFailedAt != nil ||
		unevaluatedStatus.LastEvaluatedAt != nil {
		t.Fatalf("unevaluated status = %+v, want empty membership state", unevaluatedStatus)
	}
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
