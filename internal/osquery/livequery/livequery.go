// Package livequery runs ephemeral browser-session live queries in-process.
package livequery

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrLiveQueryNotFound is returned when the manager has no live query for an id.
var ErrLiveQueryNotFound = errors.New("live query not found")

const orphanCleanupAfter = time.Minute

// Status is the per-host outcome reported back to the SSE stream.
type Status string

const (
	StatusSuccess Status = "success"
	StatusError   Status = "error"
	StatusStopped Status = "stopped"
)

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

// Event is published to subscribers for SSE delivery.
type Event struct {
	HostID   int64           `json:"host_id,omitempty"`
	HostName string          `json:"host_name,omitempty"`
	Status   Status          `json:"status"`
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// Result is one host response for a live query.
type Result struct {
	QueryID  int64
	HostID   int64
	HostName string
	Status   Status
	Data     json.RawMessage
	Error    string
}

// Manager runs ephemeral live queries entirely in-process.
type Manager struct {
	cleanupAfter time.Duration

	next    atomic.Int64
	subNext atomic.Int64

	mu        sync.Mutex
	active    map[int64]*liveQuery
	completed map[int64]struct{}
	subs      map[int64]map[int64]chan Event
}

type liveQuery struct {
	id           int64
	sql          string
	startedAt    time.Time
	pending      map[int64]struct{}
	cleanupTimer *time.Timer
}

// NewManager returns a manager for ephemeral browser-session live runs.
func NewManager() *Manager {
	return &Manager{
		cleanupAfter: orphanCleanupAfter,
		active:       make(map[int64]*liveQuery),
		completed:    make(map[int64]struct{}),
		subs:         make(map[int64]map[int64]chan Event),
	}
}

// Start registers a live query against the host set resolved when the browser
// starts the run. The returned handle is what the admin uses to attach a stream.
func (m *Manager) Start(sql string, hostIDs []int64) Handle {
	id := m.next.Add(1)
	pending := make(map[int64]struct{}, len(hostIDs))
	for _, hostID := range hostIDs {
		pending[hostID] = struct{}{}
	}
	q := &liveQuery{
		id:        id,
		sql:       sql,
		startedAt: time.Now().UTC(),
		pending:   pending,
	}

	m.mu.Lock()
	if len(pending) == 0 {
		m.completed[id] = struct{}{}
		m.mu.Unlock()
		m.forgetCompletedLater(id)
		return Handle{ID: id, SQL: sql, StartedAt: q.startedAt}
	}
	m.active[id] = q
	q.cleanupTimer = time.AfterFunc(m.cleanupAfter, func() { m.stopOrphan(id) })
	m.mu.Unlock()

	return Handle{
		ID:                id,
		SQL:               sql,
		StartedAt:         q.startedAt,
		ResolvedHostCount: int32(len(pending)),
	}
}

// PendingForHost returns live queries currently targeting host.
func (m *Manager) PendingForHost(hostID int64) []Work {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Work, 0)
	for _, q := range m.active {
		if _, pending := q.pending[hostID]; !pending {
			continue
		}
		out = append(out, Work{QueryID: q.id, SQL: q.sql})
	}
	return out
}

// Stop cancels a running live query and removes pending work from targeted
// hosts. Already-completed live queries are treated as stopped.
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
	m.stopLocked(q, StatusStopped)
	m.mu.Unlock()
	m.forgetCompletedLater(queryID)
	return nil
}

// RecordResult marks a host as having responded for a live query, publishes
// the result event, and finishes the query if no hosts remain pending.
func (m *Manager) RecordResult(result Result) {
	m.mu.Lock()
	q, ok := m.active[result.QueryID]
	if !ok {
		m.mu.Unlock()
		return
	}
	if _, pending := q.pending[result.HostID]; !pending {
		m.mu.Unlock()
		return
	}
	delete(q.pending, result.HostID)
	finished := len(q.pending) == 0
	if finished {
		m.completeLocked(q)
	}

	m.publishLocked(result.QueryID, Event{
		HostID:   result.HostID,
		HostName: result.HostName,
		Status:   result.Status,
		Data:     result.Data,
		Error:    result.Error,
	})
	if finished {
		m.closeSubscribersLocked(result.QueryID)
	}
	m.mu.Unlock()
	if finished {
		m.forgetCompletedLater(result.QueryID)
	}
}

// Subscribe returns the live event channel for queryID and a release function.
// Already-completed queries return a closed channel for late subscribers.
func (m *Manager) Subscribe(queryID int64) (<-chan Event, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.active[queryID]; ok {
		ch, release := m.subscribeLocked(queryID)
		return ch, release, nil
	}
	if _, ok := m.completed[queryID]; ok {
		ch := make(chan Event)
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
		m.stopLocked(q, StatusStopped)
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
	m.completed[q.id] = struct{}{}
}

func (m *Manager) stopLocked(q *liveQuery, status Status) {
	stopped := make([]int64, 0, len(q.pending))
	for hostID := range q.pending {
		stopped = append(stopped, hostID)
	}
	q.pending = nil
	m.completeLocked(q)

	for _, hostID := range stopped {
		m.publishLocked(q.id, Event{
			HostID: hostID,
			Status: status,
		})
	}
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

func (m *Manager) subscribeLocked(queryID int64) (<-chan Event, func()) {
	id := m.subNext.Add(1)
	ch := make(chan Event, 32)
	m.cancelCleanupLocked(queryID)

	if m.subs[queryID] == nil {
		m.subs[queryID] = make(map[int64]chan Event)
	}
	m.subs[queryID][id] = ch

	return ch, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		subs := m.subs[queryID]
		if subs == nil {
			return
		}
		if _, ok := subs[id]; !ok {
			return
		}
		delete(subs, id)
		if len(subs) == 0 {
			delete(m.subs, queryID)
			m.scheduleCleanupLocked(queryID)
		}
		close(ch)
	}
}

func (m *Manager) publishLocked(queryID int64, event Event) {
	for _, ch := range m.subs[queryID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (m *Manager) closeSubscribersLocked(queryID int64) {
	for id, ch := range m.subs[queryID] {
		close(ch)
		delete(m.subs[queryID], id)
	}
	delete(m.subs, queryID)
}
