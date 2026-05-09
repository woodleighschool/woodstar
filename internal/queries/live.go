package queries

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrLiveQueryNotFound is returned when the manager has no live query for an id.
var ErrLiveQueryNotFound = errors.New("live query not found")

// LiveQueryStatus is the per-host outcome reported back to the SSE stream.
type LiveQueryStatus string

const (
	LiveStatusSuccess LiveQueryStatus = "success"
	LiveStatusError   LiveQueryStatus = "error"
	LiveStatusTimeout LiveQueryStatus = "timeout"
)

// LiveQueryWork is one queued live query for a host (read by /distributed/read).
type LiveQueryWork struct {
	QueryID int64
	SQL     string
}

// LiveQueryHandle is the public summary of a started live query.
type LiveQueryHandle struct {
	ID                int64
	SQL               string
	StartedAt         time.Time
	ResolvedHostCount int
}

// LiveQueryEvent is published to the hub for SSE subscribers.
type LiveQueryEvent struct {
	HostID   int64           `json:"host_id,omitempty"`
	HostName string          `json:"host_name,omitempty"`
	Status   string          `json:"status"`
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// LiveQueryManager runs ephemeral live queries entirely in-process.
type LiveQueryManager struct {
	hub     *Hub
	timeout time.Duration

	next   atomic.Int64
	mu     sync.RWMutex
	active map[int64]*liveQuery
}

type liveQuery struct {
	id        int64
	sql       string
	startedAt time.Time
	pending   map[int64]struct{}
	timer     *time.Timer
	stopped   bool
}

// NewLiveQueryManager returns a manager that times out individual live queries
// after the given duration.
func NewLiveQueryManager(hub *Hub, timeout time.Duration) *LiveQueryManager {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &LiveQueryManager{
		hub:     hub,
		timeout: timeout,
		active:  make(map[int64]*liveQuery),
	}
}

// Start registers a live query against a resolved host set and arms its
// timeout. The returned handle is what the admin sees in the create response.
func (m *LiveQueryManager) Start(sql string, hostIDs []int64) *LiveQueryHandle {
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
	m.active[id] = q
	if len(pending) == 0 {
		// No targets — synthesize an immediate completion so the SSE stream
		// closes cleanly.
		m.mu.Unlock()
		m.publishCompleted(id)
		m.removeLocked(id)
		return &LiveQueryHandle{ID: id, SQL: sql, StartedAt: q.startedAt}
	}
	q.timer = time.AfterFunc(m.timeout, func() { m.expire(id) })
	m.mu.Unlock()

	return &LiveQueryHandle{
		ID:                id,
		SQL:               sql,
		StartedAt:         q.startedAt,
		ResolvedHostCount: len(hostIDs),
	}
}

// PendingForHost returns live queries currently targeting host. Live queries
// stay queued until the host responds (or the query times out), so a
// retrying agent sees the same SQL again — osquery's distributed runner is
// idempotent against re-sends.
func (m *LiveQueryManager) PendingForHost(hostID int64) []LiveQueryWork {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]LiveQueryWork, 0)
	for _, q := range m.active {
		if _, pending := q.pending[hostID]; !pending {
			continue
		}
		out = append(out, LiveQueryWork{QueryID: q.id, SQL: q.sql})
	}
	return out
}

// RecordResult marks a host as having responded for a live query, publishes
// the result event, and finishes the query if no hosts remain pending.
// Drops silently if the query has already completed or timed out — the
// admin's stream is gone, there is no replay path.
func (m *LiveQueryManager) RecordResult(
	queryID int64,
	hostID int64,
	hostName string,
	status LiveQueryStatus,
	data json.RawMessage,
	errMsg string,
) {
	m.mu.Lock()
	q, ok := m.active[queryID]
	if !ok || q.stopped {
		m.mu.Unlock()
		return
	}
	if _, pending := q.pending[hostID]; !pending {
		m.mu.Unlock()
		return
	}
	delete(q.pending, hostID)
	finished := len(q.pending) == 0
	if finished {
		q.stopped = true
		if q.timer != nil {
			q.timer.Stop()
		}
	}
	m.mu.Unlock()

	m.hub.Publish(queryID, LiveQueryEvent{
		HostID:   hostID,
		HostName: hostName,
		Status:   string(status),
		Data:     data,
		Error:    errMsg,
	})
	if finished {
		m.publishCompleted(queryID)
		m.removeLocked(queryID)
	}
}

// Subscribe returns the live event channel for queryID and a release. Errors
// with ErrLiveQueryNotFound when the query has already completed/timed out
// (no replay buffer — admins must reconnect before completion).
func (m *LiveQueryManager) Subscribe(queryID int64) (<-chan LiveQueryEvent, func(), error) {
	m.mu.RLock()
	_, ok := m.active[queryID]
	m.mu.RUnlock()
	if !ok {
		return nil, nil, ErrLiveQueryNotFound
	}
	events, release := m.hub.Subscribe(queryID)
	return events, release, nil
}

func (m *LiveQueryManager) expire(queryID int64) {
	m.mu.Lock()
	q, ok := m.active[queryID]
	if !ok || q.stopped {
		m.mu.Unlock()
		return
	}
	q.stopped = true
	timedOut := make([]int64, 0, len(q.pending))
	for hostID := range q.pending {
		timedOut = append(timedOut, hostID)
	}
	q.pending = nil
	m.mu.Unlock()

	for _, hostID := range timedOut {
		m.hub.Publish(queryID, LiveQueryEvent{
			HostID: hostID,
			Status: string(LiveStatusTimeout),
		})
	}
	m.publishCompleted(queryID)
	m.removeLocked(queryID)
}

func (m *LiveQueryManager) publishCompleted(queryID int64) {
	m.hub.Publish(queryID, LiveQueryEvent{Status: "completed"})
}

func (m *LiveQueryManager) removeLocked(queryID int64) {
	m.mu.Lock()
	delete(m.active, queryID)
	m.mu.Unlock()
}
