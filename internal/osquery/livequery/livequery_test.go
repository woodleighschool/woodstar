package livequery

import (
	"testing"
	"time"
)

func TestSubscribeReplaysPendingThenPublishesCollectedAndCloses(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []Target{{HostID: 4, HostName: "mac-4"}})

	snapshots, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	pending := receiveSnapshot(t, snapshots)
	if pending.HostID != 4 || pending.HostName != "mac-4" || pending.Status != StatusPending {
		t.Fatalf("pending snapshot = %#v, want host 4 pending", pending)
	}
	if pending.Rows == nil || len(pending.Rows) != 0 || pending.UpdatedAt.IsZero() {
		t.Fatalf("pending snapshot = %#v, want empty rows and update time", pending)
	}

	m.RecordResult(Result{
		QueryID:  handle.ID,
		HostID:   4,
		HostName: "mac-4-renamed",
		Status:   StatusCollected,
		Rows:     []map[string]string{{"answer": "1"}},
	})

	collected := receiveSnapshot(t, snapshots)
	if collected.HostID != 4 ||
		collected.HostName != "mac-4-renamed" ||
		collected.Status != StatusCollected {
		t.Fatalf("collected snapshot = %#v, want renamed host 4 collected", collected)
	}
	if len(collected.Rows) != 1 || collected.Rows[0]["answer"] != "1" {
		t.Fatalf("collected rows = %#v, want answer row", collected.Rows)
	}
	if collected.UpdatedAt.Before(pending.UpdatedAt) {
		t.Fatalf("collected update time = %s, want at or after pending %s", collected.UpdatedAt, pending.UpdatedAt)
	}

	assertClosed(t, snapshots)
}

func TestPendingForHostClearsAfterResult(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []Target{
		{HostID: 4, HostName: "mac-4"},
		{HostID: 5, HostName: "mac-5"},
	})

	if work := m.PendingForHost(4); len(work) != 1 || work[0].QueryID != handle.ID || work[0].SQL != "select 1" {
		t.Fatalf("work for host 4 = %#v, want live query work", work)
	}

	m.RecordResult(Result{QueryID: handle.ID, HostID: 4, Status: StatusCollected})

	if work := m.PendingForHost(4); len(work) != 0 {
		t.Fatalf("work for completed host = %#v, want none", work)
	}
	if work := m.PendingForHost(5); len(work) != 1 || work[0].QueryID != handle.ID {
		t.Fatalf("work for pending host = %#v, want still pending", work)
	}
}

func TestStartReportsUniqueResolvedHosts(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []Target{
		{HostID: 4, HostName: "old-name"},
		{HostID: 4, HostName: "mac-4"},
		{HostID: 5, HostName: "mac-5"},
	})

	if handle.ResolvedHostCount != 2 {
		t.Fatalf("ResolvedHostCount = %d, want unique host count 2", handle.ResolvedHostCount)
	}

	snapshots, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()
	first := receiveSnapshot(t, snapshots)
	if first.HostID != 4 || first.HostName != "mac-4" {
		t.Fatalf("first snapshot = %#v, want deduplicated current host name", first)
	}
}

func TestSubscribeCompletedEmptyQueryReceivesClosedChannel(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", nil)

	snapshots, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	assertClosed(t, snapshots)
}

func TestSubscribeReplaysCurrentStateBeforeLiveReplacements(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []Target{
		{HostID: 4, HostName: "mac-4"},
		{HostID: 5, HostName: "mac-5"},
	})
	m.RecordResult(Result{QueryID: handle.ID, HostID: 4, Status: StatusCollected})

	snapshots, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	first := receiveSnapshot(t, snapshots)
	second := receiveSnapshot(t, snapshots)
	if first.HostID != 4 || first.Status != StatusCollected {
		t.Fatalf("first replayed snapshot = %#v, want host 4 collected", first)
	}
	if second.HostID != 5 || second.Status != StatusPending {
		t.Fatalf("second replayed snapshot = %#v, want host 5 pending", second)
	}

	m.RecordResult(Result{QueryID: handle.ID, HostID: 5, Status: StatusCollected})
	live := receiveSnapshot(t, snapshots)
	if live.HostID != 5 || live.Status != StatusCollected {
		t.Fatalf("live snapshot = %#v, want host 5 collected", live)
	}
	assertClosed(t, snapshots)
}

func TestSubscribeCompletedQueryReplaysOnlyFinalSnapshots(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []Target{{HostID: 4, HostName: "mac-4"}})
	m.RecordResult(Result{
		QueryID: handle.ID,
		HostID:  4,
		Status:  StatusError,
		Error:   "query failed",
	})

	snapshots, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	snapshot := receiveSnapshot(t, snapshots)
	if snapshot.HostID != 4 || snapshot.Status != StatusError || snapshot.Error != "query failed" {
		t.Fatalf("replayed snapshot = %#v, want final error", snapshot)
	}
	assertClosed(t, snapshots)
}

func TestOrphanedRunStopsPendingHostsAfterStreamDisconnect(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []Target{
		{HostID: 4, HostName: "mac-4"},
		{HostID: 5, HostName: "mac-5"},
	})

	snapshots, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	receiveSnapshot(t, snapshots)
	receiveSnapshot(t, snapshots)
	release()
	assertClosed(t, snapshots)

	m.stopOrphan(handle.ID)
	if work := m.PendingForHost(4); len(work) != 0 {
		t.Fatalf("work for orphaned host = %#v, want none", work)
	}
	if work := m.PendingForHost(5); len(work) != 0 {
		t.Fatalf("work for orphaned host = %#v, want none", work)
	}
}

func TestStopMarksPendingHostsStoppedAndCloses(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []Target{
		{HostID: 4, HostName: "mac-4"},
		{HostID: 5, HostName: "mac-5"},
	})

	snapshots, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()
	receiveSnapshot(t, snapshots)
	receiveSnapshot(t, snapshots)

	if err := m.Stop(handle.ID); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if work := m.PendingForHost(4); len(work) != 0 {
		t.Fatalf("work for stopped host = %#v, want none", work)
	}
	if work := m.PendingForHost(5); len(work) != 0 {
		t.Fatalf("work for stopped host = %#v, want none", work)
	}

	first := receiveSnapshot(t, snapshots)
	second := receiveSnapshot(t, snapshots)
	if first.Status != StatusStopped || second.Status != StatusStopped {
		t.Fatalf("stopped snapshots = %#v %#v, want stopped", first, second)
	}
	seen := map[int64]string{first.HostID: first.HostName, second.HostID: second.HostName}
	if seen[4] != "mac-4" || seen[5] != "mac-5" {
		t.Fatalf("stopped hosts = %#v, want host names preserved", seen)
	}

	assertClosed(t, snapshots)
}

func receiveSnapshot(t *testing.T, snapshots <-chan Snapshot) Snapshot {
	t.Helper()
	select {
	case snapshot, ok := <-snapshots:
		if !ok {
			t.Fatal("snapshot channel closed")
		}
		return snapshot
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for snapshot")
		return Snapshot{}
	}
}

func assertClosed(t *testing.T, snapshots <-chan Snapshot) {
	t.Helper()
	select {
	case _, ok := <-snapshots:
		if ok {
			t.Fatal("snapshot channel remained open")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for snapshot channel to close")
	}
}
