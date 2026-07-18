package hosts

import (
	"context"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.touchByNodeKey(ctx, hostTouchSQL("orbit_node_key"), nodeKey)
}

func (s *Store) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.touchByNodeKey(ctx, hostTouchSQL("osquery_node_key"), nodeKey)
}

// SetOrbitDeviceAuthToken replaces the machine token for an Orbit node key.
func (s *Store) SetOrbitDeviceAuthToken(ctx context.Context, nodeKey, token string) error {
	tag, err := s.db.Pool().Exec(ctx, `
UPDATE hosts
SET
    orbit_device_auth_token = $2,
    last_seen_at = now(),
    updated_at = now()
WHERE orbit_node_key = $1 AND orbit_node_key <> ''`, nodeKey, token)
	if err != nil {
		return dbutil.MutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// ValidateOrbitDeviceAuthToken confirms that a machine token belongs to a host.
func (s *Store) ValidateOrbitDeviceAuthToken(ctx context.Context, token string) error {
	var hostID int64
	err := s.db.Pool().QueryRow(ctx, `
WITH touched AS (
    UPDATE hosts
    SET last_seen_at = now()
    WHERE orbit_device_auth_token = $1
      AND orbit_device_auth_token <> ''
      AND (last_seen_at IS NULL OR last_seen_at < now() - interval '1 minute')
    RETURNING id
)
SELECT id FROM touched
UNION ALL
SELECT id
FROM hosts
WHERE orbit_device_auth_token = $1
  AND orbit_device_auth_token <> ''
  AND NOT EXISTS (SELECT 1 FROM touched)
LIMIT 1`, token).Scan(&hostID)
	return dbutil.GetError(err)
}

func hostTouchSQL(nodeKeyColumn string) string {
	return `
WITH touched AS (
    UPDATE hosts
    SET last_seen_at = now()
    WHERE ` + nodeKeyColumn + ` = $1
      AND ` + nodeKeyColumn + ` <> ''
      AND (last_seen_at IS NULL OR last_seen_at < now() - interval '1 minute')
    RETURNING` + hostColumnsSQL() + `
)
SELECT` + hostColumnsSQL() + `
FROM touched
UNION ALL
SELECT` + hostColumnsSQL() + `
FROM hosts
WHERE ` + nodeKeyColumn + ` = $1
  AND ` + nodeKeyColumn + ` <> ''
  AND NOT EXISTS (SELECT 1 FROM touched)
LIMIT 1`
}

func (s *Store) touchByNodeKey(ctx context.Context, sql, nodeKey string) (*Host, error) {
	row, err := dbutil.GetOne[hostRow](ctx, s.db.Pool(), sql, nodeKey)
	if err != nil {
		return nil, err
	}
	host := hostFromRow(row, time.Now())
	return &host, nil
}
