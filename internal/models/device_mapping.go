package models

import (
	"context"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

const (
	// DeviceMappingSourceOrbitProfile is sourced from the enrollment profile.
	DeviceMappingSourceOrbitProfile = "orbit_profile"
)

// HostDeviceMapping is a user/device association observed for a host.
type HostDeviceMapping struct {
	ID        int64
	HostID    int64
	Email     string
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

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

	mappings := make([]HostDeviceMapping, 0, len(rows))
	for _, row := range rows {
		mappings = append(mappings, mappingFromRecord(row))
	}
	return mappings, nil
}

func mappingFromRecord(row sqlc.HostEmail) HostDeviceMapping {
	return HostDeviceMapping{
		ID:        row.ID,
		HostID:    row.HostID,
		Email:     row.Email,
		Source:    row.Source,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
