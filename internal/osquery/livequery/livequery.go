package livequery

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/postgres"
)

// ErrLiveQueryNotFound is returned when no retained live query has the given ID.
var ErrLiveQueryNotFound = errors.New("live query not found")

const (
	runTimeout         = 5 * time.Minute
	completedRetention = time.Minute
)

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

// Work is one unfinished live query for a host.
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

// Store persists live-query runs and per-host snapshots.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a live-query store backed by PostgreSQL.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Start creates a run against the distinct host set resolved by the caller.
func (s *Store) Start(ctx context.Context, sql string, targets []Target) (Handle, error) {
	targetsByID := make(map[int64]Target, len(targets))
	for _, target := range targets {
		targetsByID[target.HostID] = target
	}

	handle := Handle{SQL: sql}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
DELETE FROM osquery_live_query_runs
WHERE COALESCE(completed_at, deadline_at) < now() - $1 * interval '1 microsecond'`,
			completedRetention.Microseconds(),
		); err != nil {
			return err
		}

		if err := tx.QueryRow(ctx, `
INSERT INTO osquery_live_query_runs (query, deadline_at)
VALUES ($1, now() + $2 * interval '1 microsecond')
RETURNING id, started_at`, sql, runTimeout.Microseconds()).Scan(
			&handle.ID,
			&handle.StartedAt,
		); err != nil {
			return err
		}

		for _, target := range targetsByID {
			if _, err := tx.Exec(ctx, `
INSERT INTO osquery_live_query_targets (run_id, host_id, host_name)
VALUES ($1, $2, $3)`, handle.ID, target.HostID, target.HostName); err != nil {
				return err
			}
		}
		if len(targetsByID) == 0 {
			if _, err := tx.Exec(ctx, `
UPDATE osquery_live_query_runs
SET completed_at = started_at
WHERE id = $1`, handle.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Handle{}, err
	}
	handle.ResolvedHostCount = int32(len(targetsByID)) //nolint:gosec // More than MaxInt32 targets is outside supported database limits.
	return handle, nil
}

// PendingForHost returns every unfinished, unexpired run targeting hostID.
// Work remains visible until a result is recorded, so agent delivery is
// naturally at least once across replicas.
func (s *Store) PendingForHost(ctx context.Context, hostID int64) ([]Work, error) {
	qrows, err := s.pool.Query(ctx, `
SELECT run.id, run.query
FROM osquery_live_query_targets target
JOIN osquery_live_query_runs run ON run.id = target.run_id
WHERE target.host_id = $1
  AND target.status = 'pending'
  AND run.completed_at IS NULL
  AND run.deadline_at > now()
ORDER BY run.id`, hostID)
	if err != nil {
		return nil, err
	}
	work, err := pgx.CollectRows(qrows, pgx.RowToStructByPos[Work])
	if err != nil {
		return nil, err
	}
	return work, nil
}

// Stop marks every unfinished target stopped. Stopping a completed run is
// idempotent.
func (s *Store) Stop(ctx context.Context, queryID int64) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var completedAt *time.Time
		if err := tx.QueryRow(ctx, `
SELECT completed_at
FROM osquery_live_query_runs
WHERE id = $1
FOR UPDATE`, queryID).Scan(&completedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrLiveQueryNotFound
			}
			return err
		}
		if completedAt != nil {
			return nil
		}
		if _, err := tx.Exec(ctx, `
UPDATE osquery_live_query_targets
SET status = 'stopped', updated_at = now()
WHERE run_id = $1
  AND status = 'pending'`, queryID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
UPDATE osquery_live_query_runs
SET completed_at = now()
WHERE id = $1`, queryID)
		return err
	})
}

// RecordResult replaces a pending host snapshot. Duplicate, late, untargeted,
// and unknown results are ignored.
func (s *Store) RecordResult(ctx context.Context, result Result) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var completedAt *time.Time
		var expired bool
		if err := tx.QueryRow(ctx, `
SELECT completed_at, deadline_at <= now()
FROM osquery_live_query_runs
WHERE id = $1
FOR UPDATE`, result.QueryID).Scan(&completedAt, &expired); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		if completedAt != nil {
			return nil
		}
		if expired {
			return expireRun(ctx, tx, result.QueryID)
		}

		tag, err := tx.Exec(ctx, `
UPDATE osquery_live_query_targets
SET host_name = CASE WHEN $3 = '' THEN host_name ELSE $3 END,
    status = $4,
    rows = $5,
    error = $6,
    updated_at = now()
WHERE run_id = $1
  AND host_id = $2
  AND status = 'pending'`,
			result.QueryID,
			result.HostID,
			result.HostName,
			string(result.Status),
			postgres.JSONSlice[map[string]string](normalizeRows(result.Rows)),
			result.Error,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		_, err = tx.Exec(ctx, `
UPDATE osquery_live_query_runs run
SET completed_at = now()
WHERE run.id = $1
  AND NOT EXISTS (
      SELECT 1
      FROM osquery_live_query_targets target
      WHERE target.run_id = run.id
        AND target.status = 'pending'
  )`, result.QueryID)
		return err
	})
}

// Snapshots returns the current host snapshots and whether the run is
// terminal. An elapsed deadline is persisted as stopped state before return.
func (s *Store) Snapshots(ctx context.Context, queryID int64) ([]Snapshot, bool, error) {
	var snapshots []Snapshot
	var completed bool
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var completedAt *time.Time
		var expired bool
		if err := tx.QueryRow(ctx, `
SELECT completed_at, deadline_at <= now()
FROM osquery_live_query_runs
WHERE id = $1
FOR UPDATE`, queryID).Scan(&completedAt, &expired); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrLiveQueryNotFound
			}
			return err
		}
		completed = completedAt != nil
		if !completed && expired {
			if err := expireRun(ctx, tx, queryID); err != nil {
				return err
			}
			completed = true
		}

		qrows, err := tx.Query(ctx, `
SELECT host_id, host_name, status, rows, error, updated_at
FROM osquery_live_query_targets
WHERE run_id = $1
ORDER BY host_id`, queryID)
		if err != nil {
			return err
		}
		rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[snapshotRow])
		if err != nil {
			return err
		}
		snapshots = make([]Snapshot, len(rows))
		for i, row := range rows {
			snapshots[i] = Snapshot{
				HostID:    row.HostID,
				HostName:  row.HostName,
				Status:    Status(row.Status),
				Rows:      row.Rows,
				Error:     row.Error,
				UpdatedAt: row.UpdatedAt,
			}
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return snapshots, completed, nil
}

func expireRun(ctx context.Context, tx pgx.Tx, queryID int64) error {
	if _, err := tx.Exec(ctx, `
UPDATE osquery_live_query_targets target
SET status = 'stopped', updated_at = run.deadline_at
FROM osquery_live_query_runs run
WHERE target.run_id = $1
  AND run.id = target.run_id
  AND target.status = 'pending'`, queryID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
UPDATE osquery_live_query_runs
SET completed_at = deadline_at
WHERE id = $1
  AND completed_at IS NULL`, queryID)
	return err
}

func normalizeRows(rows []map[string]string) []map[string]string {
	if rows == nil {
		return []map[string]string{}
	}
	return rows
}

type snapshotRow struct {
	HostID    int64                                 `db:"host_id"`
	HostName  string                                `db:"host_name"`
	Status    string                                `db:"status"`
	Rows      postgres.JSONSlice[map[string]string] `db:"rows"`
	Error     string                                `db:"error"`
	UpdatedAt time.Time                             `db:"updated_at"`
}
