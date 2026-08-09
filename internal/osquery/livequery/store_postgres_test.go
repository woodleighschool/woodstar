//go:build postgres

package livequery

import (
	"errors"
	"sync"
	"testing"

	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestStoresShareAndIdempotentlyCompleteRun(t *testing.T) {
	db, ctx := testdb.Open(t)
	storeA := NewStore(db)
	storeB := NewStore(db)

	handle, err := storeA.Start(ctx, "select 1", []Target{
		{HostID: 4, HostName: "old-name"},
		{HostID: 4, HostName: "mac-4"},
		{HostID: 5, HostName: "mac-5"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if handle.ResolvedHostCount != 2 {
		t.Fatalf("ResolvedHostCount = %d, want 2", handle.ResolvedHostCount)
	}
	work, err := storeB.PendingForHost(ctx, 4)
	if err != nil {
		t.Fatalf("PendingForHost: %v", err)
	}
	if len(work) != 1 || work[0].QueryID != handle.ID || work[0].SQL != "select 1" {
		t.Fatalf("pending work = %+v, want run %d", work, handle.ID)
	}
	work, err = storeA.PendingForHost(ctx, 4)
	if err != nil {
		t.Fatalf("repeat PendingForHost: %v", err)
	}
	if len(work) != 1 || work[0].QueryID != handle.ID {
		t.Fatalf("repeated pending work = %+v, want run %d", work, handle.ID)
	}

	if err := storeB.RecordResult(ctx, Result{
		QueryID: handle.ID,
		HostID:  4,
		Status:  StatusCollected,
		Rows:    []map[string]string{{"answer": "first"}},
	}); err != nil {
		t.Fatalf("RecordResult: %v", err)
	}
	if err := storeA.RecordResult(ctx, Result{
		QueryID: handle.ID,
		HostID:  4,
		Status:  StatusError,
		Error:   "duplicate",
	}); err != nil {
		t.Fatalf("duplicate RecordResult: %v", err)
	}
	if err := storeA.RecordResult(ctx, Result{
		QueryID:  handle.ID,
		HostID:   5,
		HostName: "mac-5-renamed",
		Status:   StatusError,
		Error:    "query failed",
	}); err != nil {
		t.Fatalf("final RecordResult: %v", err)
	}

	snapshots, completed, err := storeB.Snapshots(ctx, handle.ID)
	if err != nil {
		t.Fatalf("Snapshots: %v", err)
	}
	if !completed || len(snapshots) != 2 {
		t.Fatalf("Snapshots = (%+v, %t), want two completed snapshots", snapshots, completed)
	}
	if snapshots[0].HostID != 4 || snapshots[0].HostName != "mac-4" ||
		snapshots[0].Status != StatusCollected || snapshots[0].Rows[0]["answer"] != "first" {
		t.Fatalf("host 4 snapshot = %+v, want first collected result", snapshots[0])
	}
	if snapshots[1].HostID != 5 || snapshots[1].HostName != "mac-5-renamed" ||
		snapshots[1].Status != StatusError || snapshots[1].Error != "query failed" {
		t.Fatalf("host 5 snapshot = %+v, want final error", snapshots[1])
	}
	work, err = storeA.PendingForHost(ctx, 4)
	if err != nil {
		t.Fatalf("PendingForHost after completion: %v", err)
	}
	if len(work) != 0 {
		t.Fatalf("pending work after completion = %+v, want none", work)
	}
}

func TestStopAndDeadlineAreSharedAcrossStores(t *testing.T) {
	db, ctx := testdb.Open(t)
	storeA := NewStore(db)
	storeB := NewStore(db)

	stopped, err := storeA.Start(ctx, "select 1", []Target{{HostID: 4, HostName: "mac-4"}})
	if err != nil {
		t.Fatalf("Start stopped run: %v", err)
	}
	if err := storeB.Stop(ctx, stopped.ID); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := storeA.Stop(ctx, stopped.ID); err != nil {
		t.Fatalf("idempotent Stop: %v", err)
	}
	snapshots, completed, err := storeA.Snapshots(ctx, stopped.ID)
	if err != nil {
		t.Fatalf("stopped Snapshots: %v", err)
	}
	if !completed || len(snapshots) != 1 || snapshots[0].Status != StatusStopped {
		t.Fatalf("stopped Snapshots = (%+v, %t), want stopped", snapshots, completed)
	}

	expired, err := storeA.Start(ctx, "select 2", []Target{{HostID: 5, HostName: "mac-5"}})
	if err != nil {
		t.Fatalf("Start expiring run: %v", err)
	}
	if _, err := db.Exec(ctx, `
UPDATE osquery_live_query_runs
SET started_at = now() - interval '2 seconds',
    deadline_at = now() - interval '1 second'
WHERE id = $1`, expired.ID); err != nil {
		t.Fatalf("expire run: %v", err)
	}
	work, err := storeB.PendingForHost(ctx, 5)
	if err != nil {
		t.Fatalf("PendingForHost expired: %v", err)
	}
	if len(work) != 0 {
		t.Fatalf("expired work = %+v, want none", work)
	}
	if err := storeB.RecordResult(ctx, Result{
		QueryID: expired.ID,
		HostID:  5,
		Status:  StatusCollected,
		Rows:    []map[string]string{{"late": "true"}},
	}); err != nil {
		t.Fatalf("late RecordResult: %v", err)
	}
	snapshots, completed, err = storeA.Snapshots(ctx, expired.ID)
	if err != nil {
		t.Fatalf("expired Snapshots: %v", err)
	}
	if !completed || len(snapshots) != 1 || snapshots[0].Status != StatusStopped || len(snapshots[0].Rows) != 0 {
		t.Fatalf("expired Snapshots = (%+v, %t), want stopped without late rows", snapshots, completed)
	}
	if err := storeA.Stop(ctx, 999999); !errors.Is(err, ErrLiveQueryNotFound) {
		t.Fatalf("missing Stop error = %v, want ErrLiveQueryNotFound", err)
	}
}

func TestConcurrentReplicaResultsCompleteRun(t *testing.T) {
	db, ctx := testdb.Open(t)
	storeA := NewStore(db)
	storeB := NewStore(db)
	handle, err := storeA.Start(ctx, "select 1", []Target{
		{HostID: 4, HostName: "mac-4"},
		{HostID: 5, HostName: "mac-5"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for store, hostID := range map[*Store]int64{storeA: 4, storeB: 5} {
		wg.Go(func() {
			errs <- store.RecordResult(ctx, Result{
				QueryID: handle.ID,
				HostID:  hostID,
				Status:  StatusCollected,
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("RecordResult: %v", err)
		}
	}

	snapshots, completed, err := storeB.Snapshots(ctx, handle.ID)
	if err != nil {
		t.Fatalf("Snapshots: %v", err)
	}
	if !completed || len(snapshots) != 2 {
		t.Fatalf("Snapshots = (%+v, %t), want completed run", snapshots, completed)
	}
}
