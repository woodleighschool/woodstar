//go:build postgres

package policies

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
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	labelA := createManualLabel(t, ctx, labelStore, "Policy A")
	labelB := createManualLabel(t, ctx, labelStore, "Policy B")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "policy-list-passing-host")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "policy-list-failing-host")
	if err := labelStore.SetMembership(ctx, labelA.ID, passingHost.ID, true); err != nil {
		t.Fatalf("include passing host: %v", err)
	}
	if err := labelStore.SetMembership(ctx, labelA.ID, failingHost.ID, true); err != nil {
		t.Fatalf("include failing host: %v", err)
	}

	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Targeted policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{labelA.ID}, []int64{labelB.ID}),
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	passes := true
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}

	got, count, err := store.List(ctx, PolicyListParams{})
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if count != 1 || len(got) != 1 {
		t.Fatalf("List returned count=%d len=%d, want one policy", count, len(got))
	}
	if got[0].PassingHostCount != 1 || got[0].FailingHostCount != 1 {
		t.Fatalf("host counts = pass %d fail %d, want 1/1", got[0].PassingHostCount, got[0].FailingHostCount)
	}
	assertTargets(t, got[0].Targets, policyTargets([]int64{labelA.ID}, []int64{labelB.ID}))
}

func TestListCountsAndSortsCurrentHostStates(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	hosts := []*hosts.Host{
		enrollTestHostDetail(t, ctx, hostStore, "policy-count-host-a"),
		enrollTestHostDetail(t, ctx, hostStore, "policy-count-host-b"),
		enrollTestHostDetail(t, ctx, hostStore, "policy-count-host-c"),
	}
	targets := policyTargets([]int64{allHostsID}, nil)
	passing, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name: "Passing count", Query: "select 1;", Targets: targets,
	}))
	if err != nil {
		t.Fatalf("create passing policy: %v", err)
	}
	failing, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name: "Failing count", Query: "select 0;", Targets: targets,
		Remediation: &PolicyRemediationMutation{Script: "#!/bin/zsh\nexit 0"},
	}))
	if err != nil {
		t.Fatalf("create failing policy: %v", err)
	}
	pending, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name: "Pending count", Query: "select 2;", Targets: targets,
		Remediation: &PolicyRemediationMutation{
			Script:    "#!/bin/zsh\nexit 0",
			Automatic: true,
		},
	}))
	if err != nil {
		t.Fatalf("create pending policy: %v", err)
	}
	for _, host := range hosts {
		recordIssuedStatus(t, ctx, store, host, passing, PolicyStatusPass, "")
	}
	recordIssuedStatus(t, ctx, store, hosts[0], failing, PolicyStatusFail, "")
	recordIssuedStatus(t, ctx, store, hosts[1], failing, PolicyStatusFail, "")
	recordIssuedStatus(t, ctx, store, hosts[2], failing, PolicyStatusError, "database locked")

	listed, _, err := store.List(ctx, PolicyListParams{})
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	byID := make(map[int64]Policy, len(listed))
	for _, policy := range listed {
		byID[policy.ID] = policy
	}
	if got := byID[passing.ID]; got.PassingHostCount != 3 || got.FailingHostCount != 0 ||
		got.ErrorHostCount != 0 || got.PendingHostCount != 0 {
		t.Fatalf("passing policy counts = %+v, want 3/0/0/0", got)
	}
	if got := byID[failing.ID]; got.PassingHostCount != 0 || got.FailingHostCount != 2 ||
		got.ErrorHostCount != 1 || got.PendingHostCount != 0 {
		t.Fatalf("failing policy counts = %+v, want 0/2/1/0", got)
	}
	if got := byID[pending.ID]; got.PassingHostCount != 0 || got.FailingHostCount != 0 ||
		got.ErrorHostCount != 0 || got.PendingHostCount != 3 {
		t.Fatalf("pending policy counts = %+v, want 0/0/0/3", got)
	}

	for sort, wantID := range map[string]int64{
		"passing_host_count.desc": passing.ID,
		"failing_host_count.desc": failing.ID,
		"error_host_count.desc":   failing.ID,
		"pending_host_count.desc": pending.ID,
		"remediation.asc":         passing.ID,
		"remediation.desc":        pending.ID,
	} {
		got, _, err := store.List(ctx, PolicyListParams{ListParams: listing.Params{Sort: sort}})
		if err != nil {
			t.Fatalf("list policies sorted by %s: %v", sort, err)
		}
		if len(got) != 3 || got[0].ID != wantID {
			t.Fatalf("policies sorted by %s = %+v, want policy %d first", sort, got, wantID)
		}
	}
}

func TestCountsAndResultsUseCurrentTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	excluded := createManualLabel(t, ctx, labelStore, "Policy excluded")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "policy-current-passing-host")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "policy-current-failing-host")
	excludedHost := enrollTestHostDetail(t, ctx, hostStore, "policy-excluded-failing-host")

	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Current target policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsID}, []int64{excluded.ID}),
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	passes := true
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), excludedHost.ID, &fails); err != nil {
		t.Fatalf("upsert excluded membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excluded.ID, excludedHost.ID, true); err != nil {
		t.Fatalf("exclude host: %v", err)
	}

	var persistedExcluded int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM osquery_policy_membership
		WHERE policy_id = $1 AND host_id = $2`,
		policy.ID,
		excludedHost.ID,
	).Scan(&persistedExcluded); err != nil {
		t.Fatalf("count excluded membership: %v", err)
	}
	if persistedExcluded != 1 {
		t.Fatalf("excluded membership rows = %d, want retained historical row", persistedExcluded)
	}
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), excludedHost.ID, &passes); err != nil {
		t.Fatalf("upsert late excluded membership: %v", err)
	}
	var persistedPasses bool
	if err := store.pool.QueryRow(ctx, `
		SELECT last_conclusive_passes
			FROM osquery_policy_membership
			WHERE policy_id = $1 AND host_id = $2`,
		policy.ID,
		excludedHost.ID,
	).Scan(&persistedPasses); err != nil {
		t.Fatalf("read excluded membership: %v", err)
	}
	if persistedPasses {
		t.Fatal("late excluded result replaced the retained historical result")
	}

	got, err := store.GetByID(ctx, policy.ID)
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if got.PassingHostCount != 1 || got.FailingHostCount != 1 {
		t.Fatalf(
			"GetByID host counts = pass %d fail %d, want 1/1",
			got.PassingHostCount,
			got.FailingHostCount,
		)
	}

	listed, _, err := store.List(ctx, PolicyListParams{})
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if len(listed) != 1 ||
		listed[0].PassingHostCount != 1 ||
		listed[0].FailingHostCount != 1 {
		t.Fatalf("List policy = %+v, want pass 1 fail 1", listed)
	}

	results, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{
		Statuses: []PolicyStatus{PolicyStatusFail},
	})
	if err != nil {
		t.Fatalf("list failing results: %v", err)
	}
	if len(results) != 1 || results[0].HostID != failingHost.ID {
		t.Fatalf("failing results = %+v, want only current failing host", results)
	}
}

func TestUpdateInvalidatesMembershipOnlyWhenQueryChanges(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-query-change-host")
	targets := policyTargets([]int64{allHostsID}, nil)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Query change policy",
		Query:   "select 1;",
		Targets: targets,
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	passes := true
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}

	metadataUpdated, err := store.Update(ctx, policy.ID, PolicyMutation{
		Name:        "Renamed policy",
		Description: "Non-semantic edit",
		Query:       " select 1; ",
		Targets:     targets,
	})
	if err != nil {
		t.Fatalf("update policy metadata: %v", err)
	}
	if testQueryHash(metadataUpdated.Query) != testQueryHash(policy.Query) {
		t.Fatal("metadata edit changed the normalized query hash")
	}
	got, err := store.GetByID(ctx, policy.ID)
	if err != nil {
		t.Fatalf("get metadata-updated policy: %v", err)
	}
	if got.PassingHostCount != 1 {
		t.Fatalf("passing count after metadata edit = %d, want 1", got.PassingHostCount)
	}

	queryUpdated, err := store.Update(ctx, policy.ID, PolicyMutation{
		Name:        "Renamed policy",
		Description: "Non-semantic edit",
		Query:       "select 2;",
		Targets:     targets,
	})
	if err != nil {
		t.Fatalf("update policy query: %v", err)
	}
	if testQueryHash(queryUpdated.Query) == testQueryHash(policy.Query) {
		t.Fatal("changed query retained its previous query hash")
	}
	got, err = store.GetByID(ctx, policy.ID)
	if err != nil {
		t.Fatalf("get query-updated policy: %v", err)
	}
	if got.PassingHostCount != 0 || got.FailingHostCount != 0 || got.PendingHostCount != 1 {
		t.Fatalf(
			"host counts after query edit = pass %d fail %d pending %d, want 0/0/1",
			got.PassingHostCount,
			got.FailingHostCount,
			got.PendingHostCount,
		)
	}
	if err := store.recordEvaluation(
		ctx,
		policy.ID,
		testQueryHash(policy.Query),
		host.ID,
		&passes,
	); err != nil {
		t.Fatalf("upsert obsolete query result: %v", err)
	}
	got, err = store.GetByID(ctx, policy.ID)
	if err != nil {
		t.Fatalf("get policy after obsolete result: %v", err)
	}
	if got.PassingHostCount != 0 || got.PendingHostCount != 1 {
		t.Fatalf(
			"counts after obsolete result = pass %d pending %d, want 0/1",
			got.PassingHostCount,
			got.PendingHostCount,
		)
	}
	if err := store.recordEvaluation(
		ctx,
		policy.ID,
		testQueryHash(queryUpdated.Query),
		host.ID,
		&passes,
	); err != nil {
		t.Fatalf("upsert current query result: %v", err)
	}
	got, err = store.GetByID(ctx, policy.ID)
	if err != nil {
		t.Fatalf("get policy after current result: %v", err)
	}
	if got.PassingHostCount != 1 || got.PendingHostCount != 0 {
		t.Fatalf(
			"counts after current result = pass %d pending %d, want 1/0",
			got.PassingHostCount,
			got.PendingHostCount,
		)
	}
}

func TestUpdatePrunesMembershipOutsideNewTargets(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	retainedLabel := createManualLabel(t, ctx, labelStore, "Policy retained")
	retainedHost := enrollTestHostDetail(t, ctx, hostStore, "policy-retained-host")
	removedHost := enrollTestHostDetail(t, ctx, hostStore, "policy-removed-host")
	if err := labelStore.SetMembership(ctx, retainedLabel.ID, retainedHost.ID, true); err != nil {
		t.Fatalf("retain host: %v", err)
	}

	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Retargeted policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	passes := true
	for _, hostID := range []int64{retainedHost.ID, removedHost.ID} {
		if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), hostID, &passes); err != nil {
			t.Fatalf("upsert host %d membership: %v", hostID, err)
		}
	}

	if _, err := store.Update(ctx, policy.ID, PolicyMutation{
		Name:    "Retargeted policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{retainedLabel.ID}, nil),
	}); err != nil {
		t.Fatalf("retarget policy: %v", err)
	}

	var hostIDs []int64
	rows, err := store.pool.Query(ctx, `
		SELECT host_id
		FROM osquery_policy_membership
		WHERE policy_id = $1
		ORDER BY host_id`,
		policy.ID,
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
	store, labelStore, _, ctx := newPostgresPolicyStore(t)
	first := createManualLabel(t, ctx, labelStore, "Policy first")
	second := createManualLabel(t, ctx, labelStore, "Policy second")
	third := createManualLabel(t, ctx, labelStore, "Policy third")

	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Replacement policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{first.ID, second.ID}, []int64{third.ID}),
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	updated, err := store.Update(ctx, policy.ID, PolicyMutation{
		Name:    "Replacement policy",
		Query:   "select 2;",
		Targets: policyTargets([]int64{third.ID}, []int64{first.ID}),
	})
	if err != nil {
		t.Fatalf("update policy: %v", err)
	}
	assertTargets(t, updated.Targets, policyTargets([]int64{third.ID}, []int64{first.ID}))

	got, err := store.GetByID(ctx, policy.ID)
	if err != nil {
		t.Fatalf("get updated policy: %v", err)
	}
	assertTargets(t, got.Targets, policyTargets([]int64{third.ID}, []int64{first.ID}))
}

func TestIssueEvaluationsForHostUsesTargetRows(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-target-host")
	matching := createManualLabel(t, ctx, labelStore, "Policy match")
	other := createManualLabel(t, ctx, labelStore, "Policy other")
	excluded := createManualLabel(t, ctx, labelStore, "Policy excluded")
	if err := labelStore.SetMembership(ctx, matching.ID, host.ID, true); err != nil {
		t.Fatalf("set matching label membership: %v", err)
	}
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Matching policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{matching.ID}, nil),
	})); err != nil {
		t.Fatalf("create matching policy: %v", err)
	}
	if _, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Nonmatching policy",
		Query:   "select 2;",
		Targets: policyTargets([]int64{other.ID}, nil),
	})); err != nil {
		t.Fatalf("create nonmatching policy: %v", err)
	}
	if _, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Excluded policy",
		Query:   "select 3;",
		Targets: policyTargets([]int64{matching.ID}, []int64{excluded.ID}),
	})); err != nil {
		t.Fatalf("create excluded policy: %v", err)
	}

	got, err := store.IssueEvaluationsForHost(ctx, host)
	if err != nil {
		t.Fatalf("issue evaluations for host: %v", err)
	}
	if len(got) != 1 || got[0].Query != "select 1;" {
		t.Fatalf("IssueEvaluationsForHost returned %+v, want only matching policy", got)
	}
}

func TestIssueEvaluationsForHostRequiresIncludeTarget(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-requires-include-host")
	excluded := createManualLabel(t, ctx, labelStore, "Policy requires include excluded")
	if err := labelStore.SetMembership(ctx, excluded.ID, host.ID, true); err != nil {
		t.Fatalf("set excluded label membership: %v", err)
	}

	if _, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Exclude-only policy",
		Query:   "select 1;",
		Targets: policyTargets(nil, []int64{excluded.ID}),
	})); err != nil {
		t.Fatalf("create exclude-only policy: %v", err)
	}

	got, err := store.IssueEvaluationsForHost(ctx, host)
	if err != nil {
		t.Fatalf("issue evaluations for host: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("IssueEvaluationsForHost returned %+v, want no policies", got)
	}
}

func TestCreatePolicyWithMissingLabelReturnsNotFound(t *testing.T) {
	store, _, _, ctx := newPostgresPolicyStore(t)

	_, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Missing label target",
		Query:   "select 1;",
		Targets: policyTargets([]int64{999_999}, nil),
	}))
	if !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("Create error = %v, want ErrNotFound", err)
	}
}

func TestHostPoliciesIncludesMatchingPolicies(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-applicable-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)

	matching, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Matching policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create matching policy: %v", err)
	}
	passes := false
	if err := store.recordEvaluation(ctx, matching.ID, testQueryHash(matching.Query), host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}

	got, _, err := store.HostPolicies(ctx, host, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("host policies: %v", err)
	}
	if len(got) != 1 || got[0].PolicyID != matching.ID {
		t.Fatalf("HostPolicies returned %+v, want matching policy", got)
	}
}

func TestHostPoliciesIncludeMembershipState(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-status-host")
	allHostsID := allHostsLabelID(t, ctx, labelStore)

	passing, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Passing policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create passing policy: %v", err)
	}
	failing, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Failing policy",
		Query:   "select 0;",
		Targets: policyTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create failing policy: %v", err)
	}
	unevaluated, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Unevaluated policy",
		Query:   "select 2;",
		Targets: policyTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create unevaluated policy: %v", err)
	}
	passes := true
	if err := store.recordEvaluation(ctx, passing.ID, testQueryHash(passing.Query), host.ID, &passes); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}
	fails := false
	if err := store.recordEvaluation(ctx, failing.ID, testQueryHash(failing.Query), host.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}

	got, _, err := store.HostPolicies(ctx, host, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("host policies: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("HostPolicies returned %d policies, want 3: %+v", len(got), got)
	}
	wantOrder := []int64{failing.ID, unevaluated.ID, passing.ID}
	for i, wantID := range wantOrder {
		if got[i].PolicyID != wantID {
			t.Fatalf("HostPolicies order = %+v, want fail/not-run/pass", got)
		}
	}
	byID := make(map[int64]PolicyHostStatus, len(got))
	for _, status := range got {
		byID[status.PolicyID] = status
	}

	passingStatus := byID[passing.ID]
	if passingStatus.Status != PolicyStatusPass {
		t.Fatalf("passing status = %q, want pass", passingStatus.Status)
	}
	if passingStatus.UpdatedAt == nil {
		t.Fatalf("passing status UpdatedAt is nil, want evaluated timestamp")
	}
	failingStatus := byID[failing.ID]
	if failingStatus.Status != PolicyStatusFail || failingStatus.UpdatedAt == nil {
		t.Fatalf("failing status = %+v, want fail with evaluated timestamp", failingStatus)
	}

	unevaluatedStatus := byID[unevaluated.ID]
	if unevaluatedStatus.Status != PolicyStatusPending || unevaluatedStatus.UpdatedAt != nil {
		t.Fatalf("unevaluated status = %+v, want pending without evaluation time", unevaluatedStatus)
	}
}

func TestPolicyResultsIncludeMembershipState(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Status list policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "aaa-failing-host")
	notRunHost := enrollTestHostDetail(t, ctx, hostStore, "bbb-not-run-host")
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "ccc-passing-host")

	fails := false
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	passes := true
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}

	got, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("policy results: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("PolicyResults returned %d hosts, want 3: %+v", len(got), got)
	}
	want := []struct {
		hostID  int64
		status  PolicyStatus
		updated bool
	}{
		{hostID: failingHost.ID, status: PolicyStatusFail, updated: true},
		{hostID: notRunHost.ID, status: PolicyStatusPending},
		{hostID: passingHost.ID, status: PolicyStatusPass, updated: true},
	}
	for i, wantStatus := range want {
		if got[i].HostID != wantStatus.hostID ||
			got[i].Status != wantStatus.status ||
			(got[i].UpdatedAt != nil) != wantStatus.updated {
			t.Fatalf(
				"PolicyResults[%d] = %+v, want host=%d status=%v updated=%v",
				i,
				got[i],
				wantStatus.hostID,
				wantStatus.status,
				wantStatus.updated,
			)
		}
	}
}

func TestPolicyResultsAndHostPoliciesSearchAndPaginate(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	firstPolicy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Alpha searchable policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create first policy: %v", err)
	}
	if _, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Bravo other policy",
		Query:   "select 2;",
		Targets: policyTargets([]int64{allHostsID}, nil),
	})); err != nil {
		t.Fatalf("create second policy: %v", err)
	}
	matchingHost := enrollTestHostDetail(t, ctx, hostStore, "policy-search-matching-host")
	otherHost := enrollTestHostDetail(t, ctx, hostStore, "policy-search-other-host")

	hostResults, count, err := store.PolicyResults(ctx, firstPolicy.ID, PolicyResultListParams{
		ListParams: listing.Params{
			Q:        "matching-host",
			PageSize: 1,
			Sort:     "host_name.desc",
		},
	})
	if err != nil {
		t.Fatalf("search policy results: %v", err)
	}
	if count != 1 || len(hostResults) != 1 || hostResults[0].HostID != matchingHost.ID {
		t.Fatalf("policy result search = %+v count=%d, want matching host", hostResults, count)
	}

	hostResults, count, err = store.PolicyResults(ctx, firstPolicy.ID, PolicyResultListParams{
		ListParams: listing.Params{PageSize: 1, Sort: "host_name.desc"},
	})
	if err != nil {
		t.Fatalf("paginate policy results: %v", err)
	}
	if count != 2 || len(hostResults) != 1 || hostResults[0].HostID != otherHost.ID {
		t.Fatalf("policy result page = %+v count=%d, want descending first of two", hostResults, count)
	}

	policyResults, count, err := store.HostPolicies(ctx, matchingHost, PolicyResultListParams{
		ListParams: listing.Params{Q: "Alpha"},
	})
	if err != nil {
		t.Fatalf("search host policies: %v", err)
	}
	if count != 1 || len(policyResults) != 1 || policyResults[0].PolicyID != firstPolicy.ID {
		t.Fatalf("host policy search = %+v count=%d, want Alpha policy", policyResults, count)
	}
}

func TestPolicyResultsFiltersByMembershipStatus(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	allHostsID := allHostsLabelID(t, ctx, labelStore)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Host ID status policy",
		Query:   "select 1",
		Targets: policyTargets([]int64{allHostsID}, nil),
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	passingHost := enrollTestHostDetail(t, ctx, hostStore, "policy-host-id-passing")
	failingHost := enrollTestHostDetail(t, ctx, hostStore, "policy-host-id-failing")
	unevaluatedHost := enrollTestHostDetail(t, ctx, hostStore, "policy-host-id-unevaluated")

	passes := true
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	if err := store.recordEvaluation(ctx, policy.ID, testQueryHash(policy.Query), unevaluatedHost.ID, nil); err != nil {
		t.Fatalf("upsert unevaluated membership: %v", err)
	}

	passingResults, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{
		Statuses: []PolicyStatus{PolicyStatusPass},
	})
	if err != nil {
		t.Fatalf("pass results: %v", err)
	}
	if len(passingResults) != 1 ||
		passingResults[0].HostID != passingHost.ID ||
		passingResults[0].HostName == "" ||
		passingResults[0].Status != PolicyStatusPass {
		t.Fatalf("pass results = %+v, want passing host status", passingResults)
	}

	failingResults, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{
		Statuses: []PolicyStatus{PolicyStatusFail},
	})
	if err != nil {
		t.Fatalf("fail results: %v", err)
	}
	if len(failingResults) != 1 ||
		failingResults[0].HostID != failingHost.ID ||
		failingResults[0].HostName == "" ||
		failingResults[0].Status != PolicyStatusFail {
		t.Fatalf("fail results = %+v, want failing host status", failingResults)
	}

	pendingResults, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{
		Statuses: []PolicyStatus{PolicyStatusPending},
	})
	if err != nil {
		t.Fatalf("pending results: %v", err)
	}
	if len(pendingResults) != 1 ||
		pendingResults[0].HostID != unevaluatedHost.ID ||
		pendingResults[0].Status != PolicyStatusPending {
		t.Fatalf("pending results = %+v, want unevaluated host status", pendingResults)
	}

	completedResults, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{
		Statuses: []PolicyStatus{PolicyStatusPass, PolicyStatusFail},
	})
	if err != nil {
		t.Fatalf("completed results: %v", err)
	}
	if len(completedResults) != 2 {
		t.Fatalf("completed results = %+v, want passing and failing hosts", completedResults)
	}
}

func TestPolicyResultsFiltersAndSortsRemediationStatuses(t *testing.T) { //nolint:gocognit // Filter matrix setup and assertions belong together.
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	hostsByStatus := map[PolicyRemediationStatusFilter]*hosts.Host{
		PolicyRemediationFilterFailed:    enrollTestHostDetail(t, ctx, hostStore, "remediation-filter-failed"),
		PolicyRemediationFilterQueued:    enrollTestHostDetail(t, ctx, hostStore, "remediation-filter-queued"),
		PolicyRemediationFilterSucceeded: enrollTestHostDetail(t, ctx, hostStore, "remediation-filter-succeeded"),
		PolicyRemediationFilterNotRun:    enrollTestHostDetail(t, ctx, hostStore, "remediation-filter-not-run"),
	}
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Remediation filter policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil),
		Remediation: &PolicyRemediationMutation{
			Script: "#!/bin/zsh\nexit 0\n",
		},
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	for _, host := range hostsByStatus {
		recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	}
	for _, status := range []PolicyRemediationStatusFilter{
		PolicyRemediationFilterFailed,
		PolicyRemediationFilterQueued,
		PolicyRemediationFilterSucceeded,
	} {
		host := hostsByStatus[status]
		makeOrbitScriptEligible(t, ctx, store, host.ID)
		summary, err := store.RunRemediations(ctx, policy.ID, []int64{host.ID}, false)
		if err != nil || summary.Queued != 1 {
			t.Fatalf("queue remediation status %q: summary=%+v err=%v", status, summary, err)
		}
		if status == PolicyRemediationFilterQueued {
			continue
		}
		run := remediationRunForTest(t, ctx, store, policy.ID, host.ID)
		exitCode := 0
		if status == PolicyRemediationFilterFailed {
			exitCode = 1
		}
		if err := store.RecordRemediationResult(ctx, host.ID, RemediationResult{
			ExecutionID: run.ExecutionID,
			ExitCode:    exitCode,
		}); err != nil {
			t.Fatalf("record remediation status %q: %v", status, err)
		}
	}

	for status, wantHost := range hostsByStatus {
		results, count, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{
			RemediationStatuses: []PolicyRemediationStatusFilter{status},
		})
		if err != nil {
			t.Fatalf("filter remediation status %q: %v", status, err)
		}
		if count != 1 || len(results) != 1 || results[0].HostID != wantHost.ID {
			t.Fatalf("filter remediation status %q = %+v count %d, want host %d", status, results, count, wantHost.ID)
		}
		if status == PolicyRemediationFilterNotRun {
			if results[0].Remediation != nil {
				t.Fatalf("not-run remediation = %+v, want nil", results[0].Remediation)
			}
		} else if results[0].Remediation == nil ||
			results[0].Remediation.Status != PolicyRemediationRunStatus(status) {
			t.Fatalf("filtered remediation = %+v, want status %q", results[0].Remediation, status)
		}
	}

	results, count, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{
		ListParams: listing.Params{Sort: "remediation.asc"},
	})
	if err != nil {
		t.Fatalf("sort remediation statuses: %v", err)
	}
	wantOrder := []PolicyRemediationStatusFilter{
		PolicyRemediationFilterFailed,
		PolicyRemediationFilterQueued,
		PolicyRemediationFilterSucceeded,
		PolicyRemediationFilterNotRun,
	}
	if count != len(wantOrder) || len(results) != len(wantOrder) {
		t.Fatalf("sorted remediation result count = %d len %d, want %d", count, len(results), len(wantOrder))
	}
	for i, status := range wantOrder {
		if results[i].HostID != hostsByStatus[status].ID {
			t.Fatalf("sorted remediation result %d host = %d, want %s host %d", i, results[i].HostID, status, hostsByStatus[status].ID)
		}
	}
}

func TestAutomaticRemediationUsesConclusiveTransitions(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-remediation-transition-host")
	makeOrbitScriptEligible(t, ctx, store, host.ID)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Remediation transition policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil),
		Remediation: &PolicyRemediationMutation{
			Script:    "#!/bin/zsh\nexit 0\n",
			Automatic: true,
		},
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	first := remediationRunForTest(t, ctx, store, policy.ID, host.ID)
	if !first.Automatic || first.Status != PolicyRemediationRunStatusQueued {
		t.Fatalf("initial remediation = %+v, want queued automatic run", first)
	}
	manual, err := store.RunRemediations(ctx, policy.ID, []int64{host.ID}, false)
	if err != nil {
		t.Fatalf("manually run queued automatic remediation: %v", err)
	}
	if manual.Queued != 0 || manual.Skipped != 1 {
		t.Fatalf("manual run summary = %+v, want existing automatic run skipped", manual)
	}
	if current := remediationRunForTest(t, ctx, store, policy.ID, host.ID); current.ExecutionID != first.ExecutionID || !current.Automatic {
		t.Fatalf("remediation after manual run = %+v, want existing automatic execution", current)
	}

	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusError, "database locked")
	errorResults, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("list error result: %v", err)
	}
	if len(errorResults) != 1 || errorResults[0].Status != PolicyStatusError || errorResults[0].Error != "database locked" {
		t.Fatalf("error result = %+v, want nonconclusive error", errorResults)
	}
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	afterError := remediationRunForTest(t, ctx, store, policy.ID, host.ID)
	if afterError.ExecutionID != first.ExecutionID {
		t.Fatalf("fail -> error -> fail queued %q, want existing %q", afterError.ExecutionID, first.ExecutionID)
	}

	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusPass, "")
	if _, err := store.RemediationRun(ctx, policy.ID, host.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("passing remediation error = %v, want ErrNotFound", err)
	}
	pending, err := store.PendingRemediationExecutionIDs(ctx, host.ID)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending remediation after pass = %v, err %v", pending, err)
	}
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	second := remediationRunForTest(t, ctx, store, policy.ID, host.ID)
	if second.ExecutionID == first.ExecutionID || second.Status != PolicyRemediationRunStatusQueued {
		t.Fatalf("pass -> fail remediation = %+v, want a new queued run", second)
	}
}

func TestEnablingAutomaticRemediationDoesNotReplayExistingFailure(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-remediation-enable-host")
	makeOrbitScriptEligible(t, ctx, store, host.ID)
	targets := policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Enable remediation policy",
		Query:   "select 1;",
		Targets: targets,
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")

	_, err = store.Update(ctx, policy.ID, PolicyMutation{
		Name:    policy.Name,
		Query:   policy.Query,
		Targets: targets,
		Remediation: &PolicyRemediationMutation{
			Script:    "#!/bin/zsh\nexit 0\n",
			Automatic: true,
		},
	})
	if err != nil {
		t.Fatalf("enable remediation: %v", err)
	}
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	if _, err := store.RemediationRun(ctx, policy.ID, host.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("remediation after enabling on failure error = %v, want ErrNotFound", err)
	}

	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusPass, "")
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	if run := remediationRunForTest(t, ctx, store, policy.ID, host.ID); !run.Automatic {
		t.Fatalf("remediation after pass -> fail = %+v, want automatic", run)
	}
}

func TestAutomaticRemediationRequiresUsableOrbitScriptExecution(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-remediation-ineligible-host")
	if _, err := store.pool.Exec(
		ctx,
		`UPDATE hosts SET orbit_scripts_enabled = true WHERE id = $1`,
		host.ID,
	); err != nil {
		t.Fatalf("record script capability without Orbit enrollment: %v", err)
	}
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Orbit eligibility policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil),
		Remediation: &PolicyRemediationMutation{
			Script:    "#!/bin/zsh\nexit 0\n",
			Automatic: true,
		},
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	if _, err := store.RemediationRun(ctx, policy.ID, host.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("automatic remediation without Orbit enrollment error = %v, want ErrNotFound", err)
	}
	summary, err := store.RunRemediations(ctx, policy.ID, []int64{host.ID}, false)
	if err != nil {
		t.Fatalf("run remediation without Orbit enrollment: %v", err)
	}
	if summary.Queued != 0 || summary.Skipped != 1 {
		t.Fatalf("ineligible run summary = %+v, want skipped", summary)
	}

	makeOrbitScriptEligible(t, ctx, store, host.ID)
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	if _, err := store.RemediationRun(ctx, policy.ID, host.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("existing failure replayed after eligibility change: %v", err)
	}
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusPass, "")
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	if run := remediationRunForTest(t, ctx, store, policy.ID, host.ID); !run.Automatic {
		t.Fatalf("eligible pass -> fail remediation = %+v, want automatic", run)
	}
}

func TestRemediationExecutionRedeliversUntilFirstResult(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-remediation-redelivery-host")
	makeOrbitScriptEligible(t, ctx, store, host.ID)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Remediation redelivery policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil),
		Remediation: &PolicyRemediationMutation{
			Script: "#!/bin/zsh\necho stable\n",
		},
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	if _, err := store.RunRemediations(ctx, policy.ID, []int64{host.ID}, false); err != nil {
		t.Fatalf("queue remediation: %v", err)
	}
	run := remediationRunForTest(t, ctx, store, policy.ID, host.ID)

	for i := range 2 {
		pending, err := store.PendingRemediationExecutionIDs(ctx, host.ID)
		if err != nil || len(pending) != 1 || pending[0] != run.ExecutionID {
			t.Fatalf("pending remediation %d = %v, err %v", i, pending, err)
		}
		execution, err := store.RemediationExecution(ctx, host.ID, run.ExecutionID)
		if err != nil {
			t.Fatalf("get remediation %d: %v", i, err)
		}
		if execution.ScriptContents != "#!/bin/zsh\necho stable\n" || execution.ExitCode != nil {
			t.Fatalf("remediation execution %d = %+v, want stable pending script", i, execution)
		}
	}

	if err := store.RecordRemediationResult(ctx, host.ID, RemediationResult{
		ExecutionID:    run.ExecutionID,
		Output:         "first",
		RuntimeSeconds: 2,
		ExitCode:       0,
	}); err != nil {
		t.Fatalf("record first result: %v", err)
	}
	if err := store.RecordRemediationResult(ctx, host.ID, RemediationResult{
		ExecutionID:    run.ExecutionID,
		Output:         "second",
		RuntimeSeconds: 3,
		ExitCode:       1,
	}); err != nil {
		t.Fatalf("record duplicate result: %v", err)
	}
	if err := store.RecordRemediationResult(ctx, host.ID, RemediationResult{
		ExecutionID: "removed-execution",
		ExitCode:    1,
	}); err != nil {
		t.Fatalf("record removed result: %v", err)
	}

	pending, err := store.PendingRemediationExecutionIDs(ctx, host.ID)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending remediation after result = %v, err %v", pending, err)
	}
	execution, err := store.RemediationExecution(ctx, host.ID, run.ExecutionID)
	if err != nil {
		t.Fatalf("get terminal remediation: %v", err)
	}
	if execution.Output != "first" || execution.RuntimeSeconds == nil ||
		*execution.RuntimeSeconds != 2 || execution.ExitCode == nil || *execution.ExitCode != 0 {
		t.Fatalf("terminal remediation = %+v, want first result", execution)
	}
	completed := remediationRunForTest(t, ctx, store, policy.ID, host.ID)
	if completed.Status != PolicyRemediationRunStatusSucceeded || completed.Output != "first" {
		t.Fatalf("completed remediation = %+v, want first successful result", completed)
	}
}

func TestTargetRemovalDropsTransientRemediation(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-remediation-retarget-host")
	makeOrbitScriptEligible(t, ctx, store, host.ID)
	include := createManualLabel(t, ctx, labelStore, "Remediation retarget include")
	if err := labelStore.SetMembership(ctx, include.ID, host.ID, true); err != nil {
		t.Fatalf("include host: %v", err)
	}
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Remediation retarget policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{include.ID}, nil),
		Remediation: &PolicyRemediationMutation{
			Script: "#!/bin/zsh\nexit 0\n",
		},
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	recordIssuedStatus(t, ctx, store, host, policy, PolicyStatusFail, "")
	if _, err := store.RunRemediations(ctx, policy.ID, []int64{host.ID}, false); err != nil {
		t.Fatalf("queue remediation: %v", err)
	}
	run := remediationRunForTest(t, ctx, store, policy.ID, host.ID)

	_, err = store.Update(ctx, policy.ID, PolicyMutation{
		Name:  policy.Name,
		Query: policy.Query,
		Remediation: &PolicyRemediationMutation{
			Script: "#!/bin/zsh\nexit 0\n",
		},
	})
	if err != nil {
		t.Fatalf("remove target: %v", err)
	}
	if _, err := store.RemediationExecution(ctx, host.ID, run.ExecutionID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("removed target execution error = %v, want ErrNotFound", err)
	}
	if err := store.RecordRemediationResult(ctx, host.ID, RemediationResult{
		ExecutionID: run.ExecutionID,
		ExitCode:    0,
	}); err != nil {
		t.Fatalf("record removed target result: %v", err)
	}
}

func TestEvaluationRejectsSupersededResultBeforeLatestCompletes(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-superseded-result-host")
	makeOrbitScriptEligible(t, ctx, store, host.ID)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Superseded result policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil),
		Remediation: &PolicyRemediationMutation{
			Script:    "#!/bin/zsh\nexit 0\n",
			Automatic: true,
		},
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	older := issuePolicyEvaluation(t, ctx, store, host, policy.ID)
	newer := issuePolicyEvaluation(t, ctx, store, host, policy.ID)
	if err := store.RecordEvaluation(
		ctx, policy.ID, testQueryHash(policy.Query), older.Revision, older.Sequence, host.ID,
		EvaluationResult{Status: PolicyStatusFail},
	); err != nil {
		t.Fatalf("record superseded failure: %v", err)
	}
	if _, err := store.RemediationRun(ctx, policy.ID, host.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("remediation after superseded failure error = %v, want ErrNotFound", err)
	}
	results, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("list result after superseded failure: %v", err)
	}
	if len(results) != 1 || results[0].Status != PolicyStatusPending {
		t.Fatalf("result after superseded failure = %+v, want pending", results)
	}

	if err := store.RecordEvaluation(
		ctx, policy.ID, testQueryHash(policy.Query), newer.Revision, newer.Sequence, host.ID,
		EvaluationResult{Status: PolicyStatusPass},
	); err != nil {
		t.Fatalf("record latest pass: %v", err)
	}
	results, _, err = store.PolicyResults(ctx, policy.ID, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("list result after latest pass: %v", err)
	}
	if len(results) != 1 || results[0].Status != PolicyStatusPass {
		t.Fatalf("result after latest pass = %+v, want pass", results)
	}
}

func TestEvaluationOrderingAndRemediationCurrentIntent(t *testing.T) { //nolint:funlen // One ordered result and remediation lifecycle.
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-remediation-order-host")
	makeOrbitScriptEligible(t, ctx, store, host.ID)
	targets := policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Ordered remediation policy",
		Query:   "select 1;",
		Targets: targets,
		Remediation: &PolicyRemediationMutation{
			Script:    "#!/bin/zsh\necho original\n",
			Automatic: true,
		},
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	older := issuePolicyEvaluation(t, ctx, store, host, policy.ID)
	newer := issuePolicyEvaluation(t, ctx, store, host, policy.ID)
	if err := store.RecordEvaluation(
		ctx, policy.ID, testQueryHash(policy.Query), newer.Revision, newer.Sequence, host.ID,
		EvaluationResult{Status: PolicyStatusFail},
	); err != nil {
		t.Fatalf("record newer failure: %v", err)
	}
	first := remediationRunForTest(t, ctx, store, policy.ID, host.ID)
	if err := store.RecordEvaluation(
		ctx, policy.ID, testQueryHash(policy.Query), older.Revision, older.Sequence, host.ID,
		EvaluationResult{Status: PolicyStatusPass},
	); err != nil {
		t.Fatalf("record stale pass: %v", err)
	}
	results, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("list result after stale pass: %v", err)
	}
	if len(results) != 1 || results[0].Status != PolicyStatusFail {
		t.Fatalf("result after stale pass = %+v, want fail", results)
	}

	execution, err := store.RemediationExecution(ctx, host.ID, first.ExecutionID)
	if err != nil {
		t.Fatalf("get original remediation: %v", err)
	}
	if execution.ScriptContents != "#!/bin/zsh\necho original\n" {
		t.Fatalf("execution script = %q, want queued snapshot", execution.ScriptContents)
	}
	policy, err = store.Update(ctx, policy.ID, PolicyMutation{
		Name:    policy.Name,
		Query:   policy.Query,
		Targets: targets,
		Remediation: &PolicyRemediationMutation{
			Script:    "#!/bin/zsh\necho edited\n",
			Automatic: true,
		},
	})
	if err != nil {
		t.Fatalf("edit remediation script: %v", err)
	}
	if _, err := store.RemediationExecution(ctx, host.ID, first.ExecutionID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("obsolete execution error = %v, want ErrNotFound", err)
	}
	if err := store.RecordRemediationResult(ctx, host.ID, RemediationResult{
		ExecutionID: first.ExecutionID,
		Output:      "done",
		ExitCode:    0,
	}); err != nil {
		t.Fatalf("record remediation result: %v", err)
	}
	if _, err := store.RemediationRun(ctx, policy.ID, host.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("obsolete completed remediation error = %v, want ErrNotFound", err)
	}
	results, _, err = store.PolicyResults(ctx, policy.ID, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("list result after remediation: %v", err)
	}
	if results[0].Status != PolicyStatusFail {
		t.Fatalf("policy status after successful script = %q, want fail", results[0].Status)
	}
	if results[0].Remediation != nil {
		t.Fatalf("obsolete remediation result = %+v, want nil", results[0].Remediation)
	}

	summary, err := store.RunRemediations(ctx, policy.ID, []int64{host.ID}, false)
	if err != nil {
		t.Fatalf("run remediation again: %v", err)
	}
	if summary.Queued != 1 || summary.Skipped != 0 {
		t.Fatalf("manual run summary = %+v, want queued", summary)
	}
	manual := remediationRunForTest(t, ctx, store, policy.ID, host.ID)
	if manual.Automatic || manual.ExecutionID == first.ExecutionID {
		t.Fatalf("manual remediation = %+v, want replacement manual run", manual)
	}
	manualExecution, err := store.RemediationExecution(ctx, host.ID, manual.ExecutionID)
	if err != nil {
		t.Fatalf("get manual remediation: %v", err)
	}
	if manualExecution.ScriptContents != "#!/bin/zsh\necho edited\n" {
		t.Fatalf("manual script = %q, want edited source", manualExecution.ScriptContents)
	}
	if err := store.RecordRemediationResult(ctx, host.ID, RemediationResult{
		ExecutionID: manual.ExecutionID,
		ExitCode:    1,
	}); err != nil {
		t.Fatalf("finish manual remediation: %v", err)
	}
}

func TestRunRemediationsQueuesEligibleFailuresAndSupersedesObsoleteWork(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	hostA := enrollTestHostDetail(t, ctx, hostStore, "policy-bulk-run-a")
	hostB := enrollTestHostDetail(t, ctx, hostStore, "policy-bulk-run-b")
	makeOrbitScriptEligible(t, ctx, store, hostA.ID)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Bulk remediation policy",
		Query:   "select 1;",
		Targets: policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil),
		Remediation: &PolicyRemediationMutation{
			Script: "#!/bin/zsh\necho original\n",
		},
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	recordIssuedStatus(t, ctx, store, hostA, policy, PolicyStatusFail, "")
	recordIssuedStatus(t, ctx, store, hostB, policy, PolicyStatusFail, "")

	summary, err := store.RunRemediations(ctx, policy.ID, nil, true)
	if err != nil {
		t.Fatalf("run remediations: %v", err)
	}
	if summary.Queued != 1 || summary.Skipped != 1 {
		t.Fatalf("run summary = %+v, want one queued and one skipped", summary)
	}
	queued := remediationRunForTest(t, ctx, store, policy.ID, hostA.ID)
	if queued.Automatic || queued.Status != PolicyRemediationRunStatusQueued {
		t.Fatalf("queued remediation = %+v, want manual queued run", queued)
	}

	summary, err = store.RunRemediations(ctx, policy.ID, nil, true)
	if err != nil {
		t.Fatalf("rerun active remediations: %v", err)
	}
	if summary.Queued != 0 || summary.Skipped != 2 {
		t.Fatalf("active run summary = %+v, want both skipped", summary)
	}
	summary, err = store.RunRemediations(
		ctx,
		policy.ID,
		[]int64{hostA.ID, hostB.ID, hostB.ID, 9_999_999},
		false,
	)
	if err != nil {
		t.Fatalf("run selected remediations: %v", err)
	}
	if summary.Queued != 0 || summary.Skipped != 3 {
		t.Fatalf("selected run summary = %+v, want active, ineligible, and stale hosts skipped", summary)
	}

	policy, err = store.Update(ctx, policy.ID, PolicyMutation{
		Name:    policy.Name,
		Query:   policy.Query,
		Targets: policy.Targets,
		Remediation: &PolicyRemediationMutation{
			Script: "#!/bin/zsh\necho edited\n",
		},
	})
	if err != nil {
		t.Fatalf("edit remediation script: %v", err)
	}
	if _, err := store.RemediationRun(ctx, policy.ID, hostA.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("obsolete remediation error = %v, want ErrNotFound", err)
	}

	summary, err = store.RunRemediations(ctx, policy.ID, []int64{hostA.ID}, false)
	if err != nil {
		t.Fatalf("run edited remediation: %v", err)
	}
	if summary.Queued != 1 || summary.Skipped != 0 {
		t.Fatalf("edited run summary = %+v, want selected host queued", summary)
	}
	edited := remediationRunForTest(t, ctx, store, policy.ID, hostA.ID)
	execution, err := store.RemediationExecution(ctx, hostA.ID, edited.ExecutionID)
	if err != nil {
		t.Fatalf("get edited remediation: %v", err)
	}
	if execution.ScriptContents != "#!/bin/zsh\necho edited\n" {
		t.Fatalf("execution script = %q, want edited script", execution.ScriptContents)
	}

	policy, err = store.Update(ctx, policy.ID, PolicyMutation{
		Name: policy.Name, Query: policy.Query, Targets: policy.Targets,
	})
	if err != nil {
		t.Fatalf("remove remediation script: %v", err)
	}
	if policy.Remediation.Configured {
		t.Fatalf("remediation summary = %+v, want no current remediation", policy.Remediation)
	}
}

func TestRunRemediationsRequiresExplicitScope(t *testing.T) {
	store := &Store{}
	ctx := context.Background()
	if _, err := store.RunRemediations(ctx, 1, nil, false); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("empty scope error = %v, want ErrInvalidInput", err)
	}
	if _, err := store.RunRemediations(ctx, 1, []int64{1}, true); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("ambiguous scope error = %v, want ErrInvalidInput", err)
	}
}

func TestEvaluationRevisionRejectsResultAfterQueryReturnsToPriorSQL(t *testing.T) {
	store, labelStore, hostStore, ctx := newPostgresPolicyStore(t)
	host := enrollTestHostDetail(t, ctx, hostStore, "policy-evaluation-revision-host")
	targets := policyTargets([]int64{allHostsLabelID(t, ctx, labelStore)}, nil)
	policy, err := store.Create(ctx, makePolicy(PolicyMutation{
		Name:    "Revision policy",
		Query:   "select 'a';",
		Targets: targets,
	}))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	stale := issuePolicyEvaluation(t, ctx, store, host, policy.ID)

	policy, err = store.Update(ctx, policy.ID, PolicyMutation{
		Name: policy.Name, Query: "select 'b';", Targets: targets,
	})
	if err != nil {
		t.Fatalf("change query to B: %v", err)
	}
	policy, err = store.Update(ctx, policy.ID, PolicyMutation{
		Name: policy.Name, Query: "select 'a';", Targets: targets,
	})
	if err != nil {
		t.Fatalf("change query back to A: %v", err)
	}
	current := issuePolicyEvaluation(t, ctx, store, host, policy.ID)
	if current.Revision <= stale.Revision {
		t.Fatalf("current revision = %d, want after stale %d", current.Revision, stale.Revision)
	}

	if err := store.RecordEvaluation(
		ctx,
		policy.ID,
		testQueryHash("select 'a';"),
		stale.Revision,
		stale.Sequence,
		host.ID,
		EvaluationResult{Status: PolicyStatusFail},
	); err != nil {
		t.Fatalf("record stale A result: %v", err)
	}
	if err := store.RecordEvaluation(
		ctx,
		policy.ID,
		testQueryHash(policy.Query),
		current.Revision,
		current.Sequence,
		host.ID,
		EvaluationResult{Status: PolicyStatusPass},
	); err != nil {
		t.Fatalf("record current A result: %v", err)
	}
	results, _, err := store.PolicyResults(ctx, policy.ID, PolicyResultListParams{})
	if err != nil {
		t.Fatalf("list policy results: %v", err)
	}
	if len(results) != 1 || results[0].Status != PolicyStatusPass {
		t.Fatalf("policy results = %+v, want current pass", results)
	}
}

func issuePolicyEvaluation(
	t *testing.T,
	ctx context.Context,
	store *Store,
	host *hosts.Host,
	policyID int64,
) Evaluation {
	t.Helper()
	evaluations, err := store.IssueEvaluationsForHost(ctx, host)
	if err != nil {
		t.Fatalf("issue evaluations: %v", err)
	}
	for _, evaluation := range evaluations {
		if evaluation.PolicyID == policyID {
			return evaluation
		}
	}
	t.Fatalf("policy %d evaluation not issued: %+v", policyID, evaluations)
	return Evaluation{}
}

func recordIssuedStatus(
	t *testing.T,
	ctx context.Context,
	store *Store,
	host *hosts.Host,
	policy *Policy,
	status PolicyStatus,
	errorMessage string,
) {
	t.Helper()
	evaluation := issuePolicyEvaluation(t, ctx, store, host, policy.ID)
	if err := store.RecordEvaluation(
		ctx,
		policy.ID,
		testQueryHash(policy.Query),
		evaluation.Revision,
		evaluation.Sequence,
		host.ID,
		EvaluationResult{Status: status, Error: errorMessage},
	); err != nil {
		t.Fatalf("record %s evaluation: %v", status, err)
	}
}

func makeOrbitScriptEligible(t *testing.T, ctx context.Context, store *Store, hostID int64) {
	t.Helper()
	if _, err := store.pool.Exec(ctx, `
		UPDATE hosts
		SET orbit_node_key = 'orbit-node-key-' || $1::text, orbit_scripts_enabled = true
		WHERE id = $1`, hostID); err != nil {
		t.Fatalf("make host Orbit script eligible: %v", err)
	}
}

func remediationRunForTest(
	t *testing.T,
	ctx context.Context,
	store *Store,
	policyID, hostID int64,
) *PolicyRemediationRun {
	t.Helper()
	run, err := store.RemediationRun(ctx, policyID, hostID)
	if err != nil {
		t.Fatalf("get remediation run: %v", err)
	}
	return run
}

func (s *Store) recordEvaluation(
	ctx context.Context,
	policyID int64,
	queryHash string,
	hostID int64,
	passes *bool,
) error {
	evaluations, err := s.IssueEvaluationsForHost(ctx, &hosts.Host{ID: hostID})
	if err != nil {
		return err
	}
	for _, evaluation := range evaluations {
		if evaluation.PolicyID != policyID {
			continue
		}
		if passes == nil {
			return nil
		}
		status := PolicyStatusFail
		if *passes {
			status = PolicyStatusPass
		}
		return s.RecordEvaluation(
			ctx,
			policyID,
			queryHash,
			evaluation.Revision,
			evaluation.Sequence,
			hostID,
			EvaluationResult{Status: status},
		)
	}
	return nil
}

func newPostgresPolicyStore(t *testing.T) (*Store, *labels.Store, *hosts.Store, context.Context) {
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

func makePolicy(m PolicyMutation) PolicyCreateMutation {
	return PolicyCreateMutation{PolicyMutation: m}
}

func policyTargets(includeIDs, excludeIDs []int64) PolicyTargets {
	return PolicyTargets{
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

func assertTargets(t *testing.T, got PolicyTargets, want PolicyTargets) {
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
