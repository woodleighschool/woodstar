//go:build postgres

package history

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestSnapshotStoresCurrentHostAndPolicyTotals(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	hostIDs := insertHistoryHosts(t, ctx, db, 4)

	observedAt := time.Date(2026, 8, 18, 10, 2, 0, 0, time.UTC)
	if _, err := db.Exec(ctx, `
		INSERT INTO host_heartbeats (host_id, source, last_seen_at)
		VALUES ($1, 'osquery', $2)`, hostIDs[0], observedAt.Add(-time.Minute)); err != nil {
		t.Fatalf("insert heartbeat: %v", err)
	}

	var policyID, allHostsLabelID int64
	if err := db.QueryRow(ctx, `
		INSERT INTO osquery_policies (name, query)
		VALUES ('History policy', 'SELECT 1')
		RETURNING id`).Scan(&policyID); err != nil {
		t.Fatalf("insert policy: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT id FROM labels WHERE builtin_key = 'all-hosts'`).Scan(&allHostsLabelID); err != nil {
		t.Fatalf("load All Hosts label: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO osquery_policy_targets (policy_id, direction, position, label_id)
		VALUES ($1, 'include', 0, $2)`, policyID, allHostsLabelID); err != nil {
		t.Fatalf("target policy: %v", err)
	}
	for _, hostID := range hostIDs {
		if _, err := db.Exec(ctx, `
			INSERT INTO label_membership (label_id, host_id) VALUES ($1, $2)`, allHostsLabelID, hostID); err != nil {
			t.Fatalf("assign host %d: %v", hostID, err)
		}
	}
	for i, status := range []string{"pass", "fail", "error"} {
		if _, err := db.Exec(ctx, `
			INSERT INTO osquery_policy_membership (policy_id, host_id, status)
			VALUES ($1, $2, $3)`, policyID, hostIDs[i], status); err != nil {
			t.Fatalf("insert %s membership: %v", status, err)
		}
	}

	if err := store.Snapshot(ctx, observedAt); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	hostPoints, err := store.ListHostStatus(ctx, observedAt.Add(-time.Hour))
	if err != nil {
		t.Fatalf("list host status: %v", err)
	}
	if len(hostPoints) != 1 || hostPoints[0].OnlineCount != 1 || hostPoints[0].OfflineCount != 3 {
		t.Fatalf("host points = %+v, want 1 online and 3 offline", hostPoints)
	}
	policyPoints, err := store.ListPolicyStatus(ctx, policyID, observedAt.Add(-time.Hour))
	if err != nil {
		t.Fatalf("list policy status: %v", err)
	}
	if len(policyPoints) != 1 || policyPoints[0].PassCount != 1 || policyPoints[0].FailCount != 1 ||
		policyPoints[0].ErrorCount != 1 || policyPoints[0].PendingCount != 1 {
		t.Fatalf("policy points = %+v, want one of each status", policyPoints)
	}

	if _, err := db.Exec(ctx, `
		UPDATE osquery_policy_membership
		SET status = 'pass'
		WHERE policy_id = $1 AND host_id = $2`, policyID, hostIDs[1]); err != nil {
		t.Fatalf("update membership: %v", err)
	}
	if err := store.Snapshot(ctx, observedAt.Add(time.Minute)); err != nil {
		t.Fatalf("replace bucket snapshot: %v", err)
	}
	policyPoints, err = store.ListPolicyStatus(ctx, policyID, observedAt.Add(-time.Hour))
	if err != nil {
		t.Fatalf("list replaced policy status: %v", err)
	}
	if len(policyPoints) != 1 || policyPoints[0].PassCount != 2 || policyPoints[0].FailCount != 0 {
		t.Fatalf("replaced policy point = %+v, want updated counts", policyPoints)
	}

	result, err := store.SweepBefore(ctx, observedAt.Truncate(BucketInterval).Add(time.Second))
	if err != nil {
		t.Fatalf("sweep history: %v", err)
	}
	if result.HostPoints != 1 || result.PolicyPoints != 1 {
		t.Fatalf("cleanup result = %+v, want one host and one policy point", result)
	}
}

func insertHistoryHosts(t *testing.T, ctx context.Context, db *pgxpool.Pool, count int) []int64 {
	t.Helper()
	hostIDs := make([]int64, count)
	for i := range hostIDs {
		if err := db.QueryRow(ctx, `
			INSERT INTO hosts (hardware_uuid, display_name)
			VALUES ($1, $2)
			RETURNING id`, fmt.Sprintf("history-host-%d", i), fmt.Sprintf("Host %d", i)).Scan(&hostIDs[i]); err != nil {
			t.Fatalf("insert host %d: %v", i, err)
		}
	}
	return hostIDs
}
