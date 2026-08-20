package history

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists and lists osquery history points.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns an osquery history store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Snapshot writes current host and policy totals into one aligned bucket.
func (s *Store) Snapshot(ctx context.Context, observedAt time.Time) error {
	observedAt = observedAt.UTC()
	bucket := observedAt.Truncate(BucketInterval)
	onlineSince := observedAt.Add(-BucketInterval)
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO osquery_host_status_points (bucket, online_count, offline_count)
			SELECT
				$1,
				count(*) FILTER (WHERE EXISTS (
					SELECT 1
					FROM host_heartbeats heartbeat
					WHERE heartbeat.host_id = hosts.id
					  AND heartbeat.source = 'osquery'
					  AND heartbeat.last_seen_at BETWEEN $2 AND $3
				))::integer,
				count(*) FILTER (WHERE NOT EXISTS (
					SELECT 1
					FROM host_heartbeats heartbeat
					WHERE heartbeat.host_id = hosts.id
					  AND heartbeat.source = 'osquery'
					  AND heartbeat.last_seen_at BETWEEN $2 AND $3
				))::integer
			FROM hosts
			ON CONFLICT (bucket) DO UPDATE SET
				online_count = EXCLUDED.online_count,
				offline_count = EXCLUDED.offline_count`, bucket, onlineSince, observedAt); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO osquery_policy_status_points (
				policy_id,
				bucket,
				pass_count,
				fail_count,
				error_count,
				pending_count
			)
			SELECT
				policy.id,
				$1,
				count(assignment.host_id) FILTER (WHERE membership.status = 'pass')::integer,
				count(assignment.host_id) FILTER (WHERE membership.status = 'fail')::integer,
				count(assignment.host_id) FILTER (WHERE membership.status = 'error')::integer,
				count(assignment.host_id) FILTER (
					WHERE membership.status IS NULL OR membership.status = 'pending'
				)::integer
			FROM osquery_policies policy
			LEFT JOIN osquery_policy_assignments assignment ON assignment.policy_id = policy.id
			LEFT JOIN osquery_policy_membership membership
				ON membership.policy_id = assignment.policy_id
			   AND membership.host_id = assignment.host_id
			GROUP BY policy.id
			ON CONFLICT (policy_id, bucket) DO UPDATE SET
				pass_count = EXCLUDED.pass_count,
				fail_count = EXCLUDED.fail_count,
				error_count = EXCLUDED.error_count,
				pending_count = EXCLUDED.pending_count`, bucket)
		return err
	})
}

// ListHostStatus returns host points from oldest to newest.
func (s *Store) ListHostStatus(ctx context.Context, since time.Time) ([]HostStatusPoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT bucket, online_count, offline_count
		FROM osquery_host_status_points
		WHERE bucket >= $1
		ORDER BY bucket`, since)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[HostStatusPoint])
}

// ListPolicyStatus returns one policy's points from oldest to newest.
func (s *Store) ListPolicyStatus(
	ctx context.Context,
	policyID int64,
	since time.Time,
) ([]PolicyStatusPoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT bucket, pass_count, fail_count, error_count, pending_count
		FROM osquery_policy_status_points
		WHERE policy_id = $1 AND bucket >= $2
		ORDER BY bucket`, policyID, since)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[PolicyStatusPoint])
}

// SweepBefore removes history older than cutoff.
func (s *Store) SweepBefore(ctx context.Context, cutoff time.Time) (CleanupResult, error) {
	var result CleanupResult
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		hosts, err := tx.Exec(ctx, `DELETE FROM osquery_host_status_points WHERE bucket < $1`, cutoff)
		if err != nil {
			return err
		}
		policies, err := tx.Exec(ctx, `DELETE FROM osquery_policy_status_points WHERE bucket < $1`, cutoff)
		if err != nil {
			return err
		}
		result.HostPoints = int(hosts.RowsAffected())
		result.PolicyPoints = int(policies.RowsAffected())
		return nil
	})
	return result, err
}
