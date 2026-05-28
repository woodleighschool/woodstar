package hosts

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

type DeviceMappingSource string

const DeviceMappingSourceOrbitProfile DeviceMappingSource = "orbit_profile"

var DeviceMappingSourceValues = []DeviceMappingSource{DeviceMappingSourceOrbitProfile}

func (DeviceMappingSource) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(DeviceMappingSourceValues...)
}

// DeviceMappingStore persists host device mappings.
type DeviceMappingStore struct {
	q *sqlc.Queries
}

func NewDeviceMappingStore(db *database.DB) *DeviceMappingStore {
	return &DeviceMappingStore{q: db.Queries()}
}

func (s *DeviceMappingStore) Upsert(ctx context.Context, hostID int64, email string, source DeviceMappingSource) error {
	if email == "" || source == "" {
		return nil
	}
	return s.q.UpsertHostDeviceMapping(ctx, sqlc.UpsertHostDeviceMappingParams{
		HostID: hostID,
		Email:  email,
		Source: string(source),
	})
}

func groupHostDeviceMappings(rows []sqlc.HostEmail, capacity int) map[int64][]HostDeviceMapping {
	grouped := make(map[int64][]HostDeviceMapping, capacity)
	for _, row := range rows {
		grouped[row.HostID] = append(grouped[row.HostID], hostDeviceMappingFromSQLC(row))
	}
	return grouped
}

func hostDeviceMappingFromSQLC(s sqlc.HostEmail) HostDeviceMapping {
	return HostDeviceMapping{
		ID:        s.ID,
		HostID:    s.HostID,
		Email:     s.Email,
		Source:    DeviceMappingSource(s.Source),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
