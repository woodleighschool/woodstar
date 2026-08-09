package mdp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
)

const (
	workerSessionTTL   = time.Minute
	workerRejectionTTL = 5 * time.Minute
)

// ErrWorkerSessionInvalid means a worker connection no longer owns the
// distribution point's current session.
var ErrWorkerSessionInvalid = errors.New("distribution point worker session is no longer current")

// ClaimWorkerSession makes connectionID the distribution point's current
// compatible worker session. A later claim replaces the existing owner.
func (s *Store) ClaimWorkerSession(
	ctx context.Context,
	pointID int64,
	key string,
	connectionID string,
	worker DistributionPointWorker,
) error {
	if connectionID == "" || !worker.Compatible || worker.ProtocolVersion == nil {
		return fmt.Errorf("%w: compatible worker session is incomplete", fault.ErrInvalidInput)
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockWorkerCredential(ctx, tx, pointID, key); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO munki_distribution_worker_sessions (
	distribution_point_id,
	connection_id,
	protocol_version,
	build_version,
	expires_at
)
VALUES ($1, $2, $3, $4, now() + $5 * interval '1 microsecond')
ON CONFLICT (distribution_point_id) DO UPDATE
SET connection_id = EXCLUDED.connection_id,
    protocol_version = EXCLUDED.protocol_version,
    build_version = EXCLUDED.build_version,
    expires_at = EXCLUDED.expires_at`,
			pointID,
			connectionID,
			*worker.ProtocolVersion,
			worker.BuildVersion,
			workerSessionTTL.Microseconds(),
		); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
DELETE FROM munki_distribution_worker_rejections
WHERE distribution_point_id = $1`, pointID)
		return err
	})
}

// ObserveRejectedWorker records the latest authenticated worker that could not
// negotiate the current protocol. A compatible session remains authoritative.
func (s *Store) ObserveRejectedWorker(
	ctx context.Context,
	pointID int64,
	key string,
	worker DistributionPointWorker,
) error {
	if worker.Compatible {
		return fmt.Errorf("%w: rejected worker cannot be compatible", fault.ErrInvalidInput)
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockWorkerCredential(ctx, tx, pointID, key); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
INSERT INTO munki_distribution_worker_rejections (
	distribution_point_id,
	protocol_version,
	build_version,
	expires_at
)
VALUES ($1, $2, $3, now() + $4 * interval '1 microsecond')
ON CONFLICT (distribution_point_id) DO UPDATE
SET protocol_version = EXCLUDED.protocol_version,
    build_version = EXCLUDED.build_version,
	expires_at = EXCLUDED.expires_at`,
			pointID,
			worker.ProtocolVersion,
			worker.BuildVersion,
			workerRejectionTTL.Microseconds(),
		)
		return err
	})
}

// RenewWorkerSession extends a current, unexpired compatible session using
// PostgreSQL time.
func (s *Store) RenewWorkerSession(
	ctx context.Context,
	pointID int64,
	connectionID string,
) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE munki_distribution_worker_sessions session
SET expires_at = now() + $3 * interval '1 microsecond'
FROM munki_distribution_points point
WHERE session.distribution_point_id = $1
  AND session.connection_id = $2
  AND session.expires_at > now()
  AND point.id = session.distribution_point_id
  AND point.enabled`, pointID, connectionID, workerSessionTTL.Microseconds())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrWorkerSessionInvalid
	}
	return nil
}

// ReleaseWorkerSession removes connectionID only while it remains the current
// owner. A superseded connection cannot release its replacement's session.
func (s *Store) ReleaseWorkerSession(
	ctx context.Context,
	pointID int64,
	connectionID string,
) error {
	_, err := s.pool.Exec(ctx, `
DELETE FROM munki_distribution_worker_sessions
WHERE distribution_point_id = $1
  AND connection_id = $2`, pointID, connectionID)
	return err
}

func lockWorkerCredential(
	ctx context.Context,
	tx pgx.Tx,
	pointID int64,
	key string,
) error {
	var found bool
	err := tx.QueryRow(ctx, `
SELECT true
FROM munki_distribution_points
WHERE id = $1
  AND enabled
  AND "key" = $2
FOR SHARE`, pointID, key).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkerSessionInvalid
	}
	return err
}

func clearWorkerStates(ctx context.Context, tx pgx.Tx, pointID int64) error {
	if _, err := tx.Exec(ctx, `
DELETE FROM munki_distribution_worker_sessions
WHERE distribution_point_id = $1`, pointID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
DELETE FROM munki_distribution_worker_rejections
WHERE distribution_point_id = $1`, pointID)
	return err
}
