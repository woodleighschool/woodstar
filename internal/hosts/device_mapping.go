package hosts

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// DeviceMappingSourceOrbitProfile is sourced from the enrollment profile.
const DeviceMappingSourceOrbitProfile = "orbit_profile"

// DeviceMappingStore persists host device mappings.
type DeviceMappingStore struct {
	q *sqlc.Queries
}

// NewDeviceMappingStore returns a device mapping store backed by db.
func NewDeviceMappingStore(db *database.DB) *DeviceMappingStore {
	return &DeviceMappingStore{q: db.Queries()}
}

// Upsert stores the latest email for a source.
func (s *DeviceMappingStore) Upsert(ctx context.Context, hostID int64, email, source string) error {
	if email == "" || source == "" {
		return nil
	}
	return s.q.UpsertHostDeviceMapping(ctx, sqlc.UpsertHostDeviceMappingParams{
		HostID: hostID,
		Email:  email,
		Source: source,
	})
}

// ListForHost returns mappings in stable source order.
func (s *DeviceMappingStore) ListForHost(ctx context.Context, hostID int64) ([]HostDeviceMapping, error) {
	rows, err := s.q.ListHostDeviceMappings(ctx, sqlc.ListHostDeviceMappingsParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	mappings := make([]HostDeviceMapping, len(rows))
	for i, row := range rows {
		mappings[i] = hostDeviceMappingFromSQLC(row)
	}
	return mappings, nil
}

// ListForHosts returns mappings grouped by host_id for bulk list responses.
// Hosts with no mappings get a nil slice in the result map (omit on the wire).
func (s *DeviceMappingStore) ListForHosts(ctx context.Context, hostIDs []int64) (map[int64][]HostDeviceMapping, error) {
	if len(hostIDs) == 0 {
		return map[int64][]HostDeviceMapping{}, nil
	}
	rows, err := s.q.ListHostDeviceMappingsForHosts(ctx, sqlc.ListHostDeviceMappingsForHostsParams{HostIds: hostIDs})
	if err != nil {
		return nil, err
	}
	grouped := make(map[int64][]HostDeviceMapping, len(hostIDs))
	for _, row := range rows {
		grouped[row.HostID] = append(grouped[row.HostID], hostDeviceMappingFromSQLC(row))
	}
	return grouped, nil
}

func hostDeviceMappingFromSQLC(s sqlc.HostEmail) HostDeviceMapping {
	return HostDeviceMapping{
		ID:        s.ID,
		HostID:    s.HostID,
		Email:     s.Email,
		Source:    s.Source,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
