package models

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// DeviceMappingSourceOrbitProfile is sourced from the enrollment profile.
const DeviceMappingSourceOrbitProfile = "orbit_profile"

// HostDeviceMapping is a user/device association observed for a host. Stored
// in the host_emails table; the alias keeps the domain term in handler code.
type HostDeviceMapping = sqlc.HostEmail

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
	return s.q.ListHostDeviceMappings(ctx, sqlc.ListHostDeviceMappingsParams{HostID: hostID})
}
