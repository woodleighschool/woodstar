package hosts

import (
	"context"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

// OsqueryHost contains the host state used by the osquery protocol.
type OsqueryHost struct {
	ID                        int64      `db:"id"`
	DisplayName               string     `db:"display_name"`
	InventoryUpdatedAt        *time.Time `db:"inventory_updated_at"`
	InventoryQueryHash        string     `db:"inventory_query_hash"`
	InventoryRefreshRequested bool       `db:"inventory_refresh_requested"`
}

func (s *Store) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*OsqueryHost, error) {
	host, err := postgres.GetOne[OsqueryHost](ctx, s.pool, `
SELECT id, display_name, inventory_updated_at, inventory_query_hash, inventory_refresh_requested
FROM hosts WHERE osquery_node_key = $1 AND osquery_node_key <> ''`, nodeKey)
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (s *Store) IDByOrbitNodeKey(ctx context.Context, nodeKey string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM hosts WHERE orbit_node_key = $1 AND orbit_node_key <> ''`, nodeKey).Scan(&id)
	return id, postgres.GetError(err)
}

func (s *Store) IDByHardwareSerial(ctx context.Context, serial string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM hosts WHERE hardware_serial = $1 AND hardware_serial <> ''`, strings.TrimSpace(serial)).Scan(&id)
	return id, postgres.GetError(err)
}

// SetOrbitDeviceAuthToken replaces the machine token for an Orbit node key.
func (s *Store) SetOrbitDeviceAuthToken(ctx context.Context, nodeKey, token string) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE hosts
SET
	orbit_device_auth_token = $2,
	updated_at = now()
WHERE orbit_node_key = $1 AND orbit_node_key <> ''`, nodeKey, token)
	if err != nil {
		return postgres.MutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

func (s *Store) IDByOrbitDeviceAuthToken(ctx context.Context, token string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM hosts WHERE orbit_device_auth_token = $1 AND orbit_device_auth_token <> ''`, token).Scan(&id)
	return id, postgres.GetError(err)
}
