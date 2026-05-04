package models

import (
	"context"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
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
	db *database.DB
}

// NewDeviceMappingStore returns a device mapping store backed by db.
func NewDeviceMappingStore(db *database.DB) *DeviceMappingStore {
	return &DeviceMappingStore{db: db}
}

// Upsert stores the latest email for a source.
func (s *DeviceMappingStore) Upsert(ctx context.Context, hostID int64, email, source string) error {
	if email == "" || source == "" {
		return nil
	}
	return s.db.Exec(ctx, `
INSERT INTO host_emails (host_id, email, source)
VALUES ($1, $2, $3)
ON CONFLICT (host_id, source) DO UPDATE SET
    email = EXCLUDED.email,
    updated_at = now()`,
		hostID, email, source,
	)
}

// ListForHost returns mappings in stable source order.
func (s *DeviceMappingStore) ListForHost(ctx context.Context, hostID int64) ([]HostDeviceMapping, error) {
	rows, err := s.db.Query(ctx, `
SELECT id, host_id, email, source, created_at, updated_at
FROM host_emails
WHERE host_id = $1
ORDER BY source`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mappings := make([]HostDeviceMapping, 0)
	for rows.Next() {
		var mapping HostDeviceMapping
		if err := rows.Scan(
			&mapping.ID,
			&mapping.HostID,
			&mapping.Email,
			&mapping.Source,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		); err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}
	return mappings, rows.Err()
}
