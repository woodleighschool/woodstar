package hosts

import (
	"context"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.getByIdentity(ctx, "orbit_node_key", nodeKey)
}

func (s *Store) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.getByIdentity(ctx, "osquery_node_key", nodeKey)
}

// SetOrbitDeviceAuthToken replaces the machine token for an Orbit node key.
func (s *Store) SetOrbitDeviceAuthToken(ctx context.Context, nodeKey, token string) error {
	tag, err := s.db.Pool().Exec(ctx, `
UPDATE hosts
SET
	orbit_device_auth_token = $2,
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
func (s *Store) ValidateOrbitDeviceAuthToken(ctx context.Context, token string) (*Host, error) {
	row, err := dbutil.GetOne[hostRow](ctx, s.db.Pool(), hostSelectSQL()+`
WHERE orbit_device_auth_token = $1
  AND orbit_device_auth_token <> ''`, token)
	if err != nil {
		return nil, err
	}
	host := hostFromRow(row, time.Now())
	return &host, nil
}

func (s *Store) getByIdentity(ctx context.Context, column, value string) (*Host, error) {
	row, err := dbutil.GetOne[hostRow](ctx, s.db.Pool(), hostSelectSQL()+`
WHERE `+column+` = $1
  AND `+column+` <> ''`, value)
	if err != nil {
		return nil, err
	}
	host := hostFromRow(row, time.Now())
	return &host, nil
}
