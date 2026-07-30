// Package livequery runs ephemeral browser-session live queries in-process.
package livequery

import (
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ErrLiveQueryNotFound is returned when the manager has no live query for an id.
var ErrLiveQueryNotFound = errors.New("live query not found")

const orphanCleanupAfter = time.Minute

// Status is one host's state within a live query.
type Status string

const (
	StatusPending   Status = "pending"
	StatusCollected Status = "collected"
	StatusError     Status = "error"
	StatusStopped   Status = "stopped"
)

// Target is one resolved online host at the start of a live query.
type Target struct {
	HostID   int64
	HostName string
}

// Work is one queued live query for a host (read by /distributed/read).
type Work struct {
	QueryID int64
	SQL     string
}

// Handle is the public summary of a started live query.
type Handle struct {
	ID                int64     `json:"id"`
	SQL               string    `json:"sql"`
	StartedAt         time.Time `json:"started_at"`
	ResolvedHostCount int32     `json:"resolved_host_count"`
}

// Snapshot is the current state of one host in a live query.
type Snapshot struct {
	HostID    int64
	HostName  string
	Status    Status
	Rows      []map[string]string
	Error     string
	UpdatedAt time.Time
}

// Result is one host response for a live query.
type Result struct {
	QueryID  int64
	HostID   int64
	HostName string
	Status   Status
	Rows     []map[string]string
	Error    string
}

// Manager runs ephemeral live queries entirely in-process.
type Manager struct {
	cleanupAfter time.Duration

	next    atomic.Int64
	subNext atomic.Int64

	mu        sync.Mutex
	active    map[int64]*liveQuery
	completed map[int64]*liveQuery
	subs      map[int64]map[int64]chan Snapshot
}

type liveQuery struct {
	id           int64
	sql          string
	startedAt    time.Time
	snapshots    map[int64]Snapshot
	cleanupTimer *time.Timer
}

// NewManager returns a manager for ephemeral browser-session live runs.
func NewManager() *Manager {
	return &Manager{
		cleanupAfter: orphanCleanupAfter,
		active:       make(map[int64]*liveQuery),
		completed:    make(map[int64]*liveQuery),
		subs:         make(map[int64]map[int64]chan Snapshot),
	}
}

// Start registers a live query against the host set resolved when the browser
// starts the run. The returned handle is what the admin uses to attach a stream.
func (m *Manager) Start(sql string, targets []Target) Handle {
	id := m.next.Add(1)
	startedAt := time.Now().UTC()
	snapshots := make(map[int64]Snapshot, len(targets))
	for _, target := range targets {
		snapshots[target.HostID] = Snapshot{
			HostID:    target.HostID,
			HostName:  target.HostName,
			Status:    StatusPending,
			Rows:      []map[string]string{},
			UpdatedAt: startedAt,
		}
	}
	q := &liveQuery{
		id:        id,
		sql:       sql,
		startedAt: startedAt,
		snapshots: snapshots,
	}

	m.mu.Lock()
	if len(snapshots) == 0 {
		m.completed[id] = q
		m.mu.Unlock()
		m.forgetCompletedLater(id)
		return Handle{ID: id, SQL: sql, StartedAt: startedAt}
	}
	m.active[id] = q
	q.cleanupTimer = time.AfterFunc(m.cleanupAfter, func() { m.stopOrphan(id) })
	m.mu.Unlock()

	return Handle{
		ID:                id,
		SQL:               sql,
		StartedAt:         startedAt,
		ResolvedHostCount: int32(len(snapshots)), //nolint:gosec // More than MaxInt32 distinct in-memory hosts is outside supported process limits.
	}
}

// PendingForHost returns live queries currently targeting host.
func (m *Manager) PendingForHost(hostID int64) []Work {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Work, 0)
	for _, q := range m.active {
		snapshot, targeted := q.snapshots[hostID]
		if !targeted || snapshot.Status != StatusPending {
			continue
		}
		out = append(out, Work{QueryID: q.id, SQL: q.sql})
	}
	return out
}

// Stop cancels a running live query and marks its pending hosts stopped.
// Stopping an already-completed live query is idempotent.
func (m *Manager) Stop(queryID int64) error {
	m.mu.Lock()
	q, ok := m.active[queryID]
	if !ok {
		if _, completed := m.completed[queryID]; completed {
			m.mu.Unlock()
			return nil
		}
		m.mu.Unlock()
		return ErrLiveQueryNotFound
	}
	m.stopLocked(q)
	m.mu.Unlock()
	m.forgetCompletedLater(queryID)
	return nil
}

// RecordResult replaces a host's pending snapshot with its response and
// finishes the query when no hosts remain pending.
func (m *Manager) RecordResult(result Result) {
	m.mu.Lock()
	q, ok := m.active[result.QueryID]
	if !ok {
		m.mu.Unlock()
		return
	}
	snapshot, targeted := q.snapshots[result.HostID]
	if !targeted || snapshot.Status != StatusPending {
		m.mu.Unlock()
		return
	}
	if result.HostName != "" {
		snapshot.HostName = result.HostName
	}
	snapshot.Status = result.Status
	snapshot.Rows = normalizeRows(result.Rows)
	snapshot.Error = result.Error
	snapshot.UpdatedAt = time.Now().UTC()
	q.snapshots[result.HostID] = snapshot
	m.publishLocked(result.QueryID, snapshot)

	finished := !hasPendingSnapshots(q)
	if finished {
		m.completeLocked(q)
		m.closeSubscribersLocked(result.QueryID)
	}
	m.mu.Unlock()
	if finished {
		m.forgetCompletedLater(result.QueryID)
	}
}

// Subscribe returns current host snapshots followed by live replacements and a
// release function. Completed queries replay their final snapshots and close.
func (m *Manager) Subscribe(queryID int64) (<-chan Snapshot, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if q, ok := m.active[queryID]; ok {
		ch, release := m.subscribeLocked(q)
		return ch, release, nil
	}
	if q, ok := m.completed[queryID]; ok {
		ch := make(chan Snapshot, len(q.snapshots))
		m.replayLocked(q, ch)
		close(ch)
		return ch, func() {}, nil
	}
	return nil, nil, ErrLiveQueryNotFound
}

func (m *Manager) stopOrphan(queryID int64) {
	m.mu.Lock()
	if len(m.subs[queryID]) > 0 {
		m.mu.Unlock()
		return
	}
	stopped := false
	if q, ok := m.active[queryID]; ok {
		m.stopLocked(q)
		stopped = true
	}
	m.mu.Unlock()
	if stopped {
		m.forgetCompletedLater(queryID)
	}
}

func (m *Manager) completeLocked(q *liveQuery) {
	if q.cleanupTimer != nil {
		q.cleanupTimer.Stop()
	}
	delete(m.active, q.id)
	m.completed[q.id] = q
}

func (m *Manager) stopLocked(q *liveQuery) {
	updatedAt := time.Now().UTC()
	for hostID, snapshot := range q.snapshots {
		if snapshot.Status != StatusPending {
			continue
		}
		snapshot.Status = StatusStopped
		snapshot.UpdatedAt = updatedAt
		q.snapshots[hostID] = snapshot
		m.publishLocked(q.id, snapshot)
	}
	m.completeLocked(q)
	m.closeSubscribersLocked(q.id)
}

func (m *Manager) scheduleCleanupLocked(queryID int64) {
	q, ok := m.active[queryID]
	if !ok {
		return
	}
	if q.cleanupTimer != nil {
		q.cleanupTimer.Stop()
	}
	q.cleanupTimer = time.AfterFunc(m.cleanupAfter, func() { m.stopOrphan(queryID) })
}

func (m *Manager) cancelCleanupLocked(queryID int64) {
	q, ok := m.active[queryID]
	if !ok || q.cleanupTimer == nil {
		return
	}
	q.cleanupTimer.Stop()
	q.cleanupTimer = nil
}

func (m *Manager) forgetCompletedLater(queryID int64) {
	time.AfterFunc(m.cleanupAfter, func() {
		m.mu.Lock()
		delete(m.completed, queryID)
		m.mu.Unlock()
	})
}

func (m *Manager) subscribeLocked(q *liveQuery) (<-chan Snapshot, func()) {
	id := m.subNext.Add(1)
	// One initial and one terminal snapshot per host can be queued without
	// blocking the producer before the stream consumer starts.
	ch := make(chan Snapshot, len(q.snapshots)*2)
	m.replayLocked(q, ch)
	m.cancelCleanupLocked(q.id)

	if m.subs[q.id] == nil {
		m.subs[q.id] = make(map[int64]chan Snapshot)
	}
	m.subs[q.id][id] = ch

	return ch, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		subs := m.subs[q.id]
		if subs == nil {
			return
		}
		if _, ok := subs[id]; !ok {
			return
		}
		delete(subs, id)
		if len(subs) == 0 {
			delete(m.subs, q.id)
			m.scheduleCleanupLocked(q.id)
		}
		close(ch)
	}
}

func (m *Manager) replayLocked(q *liveQuery, ch chan<- Snapshot) {
	snapshots := make([]Snapshot, 0, len(q.snapshots))
	for _, snapshot := range q.snapshots {
		snapshots = append(snapshots, snapshot)
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].HostID < snapshots[j].HostID
	})
	for _, snapshot := range snapshots {
		ch <- snapshot
	}
}

func (m *Manager) publishLocked(queryID int64, snapshot Snapshot) {
	for _, ch := range m.subs[queryID] {
		ch <- snapshot
	}
}

func (m *Manager) closeSubscribersLocked(queryID int64) {
	for id, ch := range m.subs[queryID] {
		close(ch)
		delete(m.subs[queryID], id)
	}
	delete(m.subs, queryID)
}

func hasPendingSnapshots(q *liveQuery) bool {
	for _, snapshot := range q.snapshots {
		if snapshot.Status == StatusPending {
			return true
		}
	}
	return false
}

func normalizeRows(rows []map[string]string) []map[string]string {
	if rows == nil {
		return []map[string]string{}
	}
	return rows
}
