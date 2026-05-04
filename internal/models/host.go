package models

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
)

// Host is an enrolled Mac.
type Host struct {
	ID               int64
	HardwareUUID     string
	DisplayName      string
	Hostname         string
	ComputerName     string
	HardwareSerial   string
	HardwareModel    string
	Platform         string
	PlatformLike     string
	OSVersion        string
	OsqueryVersion   string
	OrbitVersion     string
	OrbitNodeKey     string
	OsqueryNodeKey   string
	CPUBrand         string
	CPULogicalCores  int
	CPUPhysicalCores int
	PhysicalMemory   int64
	HardwareVendor   string
	KernelVersion    string
	EnrolledAt       *time.Time
	LastSeenAt       *time.Time
	DetailUpdatedAt  *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// HostStore persists Orbit-managed Macs.
type HostStore struct {
	db *database.DB
}

// NewHostStore returns a host store backed by db.
func NewHostStore(db *database.DB) *HostStore {
	return &HostStore{db: db}
}

// EnrollParams holds the fields supplied by an Orbit enrollment request.
// Only HardwareUUID and OrbitNodeKey are required; the rest are best-effort hints.
type EnrollParams struct {
	HardwareUUID   string
	HardwareSerial string
	Hostname       string
	ComputerName   string
	HardwareModel  string
	Platform       string
	PlatformLike   string
	OrbitNodeKey   string
}

// HostDetailUpdate is inventory reported by osquery detail queries.
type HostDetailUpdate struct {
	HardwareUUID     string
	Hostname         string
	ComputerName     string
	HardwareSerial   string
	HardwareModel    string
	OSVersion        string
	Platform         string
	PlatformLike     string
	KernelVersion    string
	HardwareVendor   string
	CPUBrand         string
	CPULogicalCores  int
	CPUPhysicalCores int
	PhysicalMemory   int64
	OsqueryVersion   string
	OsqueryNodeKey   string
}

const upsertOsqueryEnrollSQL = `
INSERT INTO hosts (
    hardware_uuid, display_name, hostname, computer_name, hardware_serial,
    hardware_model, os_version, platform, platform_like, osquery_version,
    osquery_node_key, cpu_brand, cpu_logical_cores, cpu_physical_cores,
    physical_memory, hardware_vendor, kernel_version, last_seen_at, detail_updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
        $11, $12, $13, $14, $15, $16, $17, now(), NULL)
ON CONFLICT (hardware_uuid) DO UPDATE SET
    display_name        = EXCLUDED.display_name,
    hostname            = EXCLUDED.hostname,
    computer_name       = EXCLUDED.computer_name,
    hardware_serial     = EXCLUDED.hardware_serial,
    hardware_model      = EXCLUDED.hardware_model,
    os_version          = EXCLUDED.os_version,
    platform            = EXCLUDED.platform,
    platform_like       = EXCLUDED.platform_like,
    osquery_version     = EXCLUDED.osquery_version,
    osquery_node_key    = EXCLUDED.osquery_node_key,
    cpu_brand           = EXCLUDED.cpu_brand,
    cpu_logical_cores   = EXCLUDED.cpu_logical_cores,
    cpu_physical_cores  = EXCLUDED.cpu_physical_cores,
    physical_memory     = EXCLUDED.physical_memory,
    hardware_vendor     = EXCLUDED.hardware_vendor,
    kernel_version      = EXCLUDED.kernel_version,
    detail_updated_at   = NULL,
    last_seen_at        = now(),
    updated_at          = now(),
    deleted_at          = NULL
RETURNING id, hardware_uuid, display_name, hostname, computer_name,
          hardware_serial, hardware_model, platform, platform_like,
          os_version, osquery_version, orbit_version,
          COALESCE(orbit_node_key, ''), COALESCE(osquery_node_key, ''),
          cpu_brand, cpu_logical_cores, cpu_physical_cores, physical_memory,
          hardware_vendor, kernel_version,
          enrolled_at, last_seen_at, detail_updated_at,
          created_at, updated_at`

// UpsertOnOrbitEnroll inserts a new host or refreshes an existing one keyed by hardware UUID.
// Re-enrollment overwrites the node key so prior keys stop authenticating.
func (s *HostStore) UpsertOnOrbitEnroll(ctx context.Context, params EnrollParams) (*Host, error) {
	if strings.TrimSpace(params.HardwareUUID) == "" {
		return nil, errors.New("hardware_uuid is required")
	}
	if strings.TrimSpace(params.OrbitNodeKey) == "" {
		return nil, errors.New("orbit_node_key is required")
	}

	host := &Host{}
	err := s.db.QueryRow(ctx, `
INSERT INTO hosts (
    hardware_uuid, display_name, hostname, computer_name,
    hardware_serial, hardware_model, platform, platform_like,
    orbit_node_key, enrolled_at, last_seen_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())
ON CONFLICT (hardware_uuid) DO UPDATE SET
    display_name    = EXCLUDED.display_name,
    hostname        = EXCLUDED.hostname,
    computer_name   = EXCLUDED.computer_name,
    hardware_serial = EXCLUDED.hardware_serial,
    hardware_model  = EXCLUDED.hardware_model,
    platform        = EXCLUDED.platform,
    platform_like   = EXCLUDED.platform_like,
    orbit_node_key  = EXCLUDED.orbit_node_key,
    enrolled_at     = now(),
    last_seen_at    = now(),
    updated_at      = now(),
    deleted_at      = NULL
RETURNING id, hardware_uuid, display_name, hostname, computer_name,
          hardware_serial, hardware_model, platform, platform_like,
          os_version, osquery_version, orbit_version,
          COALESCE(orbit_node_key, ''), COALESCE(osquery_node_key, ''),
          cpu_brand, cpu_logical_cores, cpu_physical_cores, physical_memory,
          hardware_vendor, kernel_version,
          enrolled_at, last_seen_at, detail_updated_at,
          created_at, updated_at`,
		params.HardwareUUID,
		displayNameFor(params),
		params.Hostname,
		params.ComputerName,
		params.HardwareSerial,
		params.HardwareModel,
		params.Platform,
		params.PlatformLike,
		params.OrbitNodeKey,
	).Scan(
		&host.ID,
		&host.HardwareUUID,
		&host.DisplayName,
		&host.Hostname,
		&host.ComputerName,
		&host.HardwareSerial,
		&host.HardwareModel,
		&host.Platform,
		&host.PlatformLike,
		&host.OSVersion,
		&host.OsqueryVersion,
		&host.OrbitVersion,
		&host.OrbitNodeKey,
		&host.OsqueryNodeKey,
		&host.CPUBrand,
		&host.CPULogicalCores,
		&host.CPUPhysicalCores,
		&host.PhysicalMemory,
		&host.HardwareVendor,
		&host.KernelVersion,
		&host.EnrolledAt,
		&host.LastSeenAt,
		&host.DetailUpdatedAt,
		&host.CreatedAt,
		&host.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return host, nil
}

// UpsertOnOsqueryEnroll refreshes the osquery node key and rich host inventory.
func (s *HostStore) UpsertOnOsqueryEnroll(ctx context.Context, update HostDetailUpdate) (*Host, error) {
	update.HardwareUUID = strings.TrimSpace(update.HardwareUUID)
	update.OsqueryNodeKey = strings.TrimSpace(update.OsqueryNodeKey)
	if update.HardwareUUID == "" {
		return nil, errors.New("hardware_uuid is required")
	}
	if update.OsqueryNodeKey == "" {
		return nil, errors.New("osquery_node_key is required")
	}

	host := &Host{}
	err := s.db.QueryRow(ctx, upsertOsqueryEnrollSQL,
		update.HardwareUUID,
		displayNameFor(EnrollParams{
			HardwareUUID: update.HardwareUUID,
			Hostname:     update.Hostname,
			ComputerName: update.ComputerName,
		}),
		update.Hostname,
		update.ComputerName,
		update.HardwareSerial,
		update.HardwareModel,
		update.OSVersion,
		update.Platform,
		update.PlatformLike,
		update.OsqueryVersion,
		update.OsqueryNodeKey,
		update.CPUBrand,
		update.CPULogicalCores,
		update.CPUPhysicalCores,
		update.PhysicalMemory,
		update.HardwareVendor,
		update.KernelVersion,
	).Scan(
		&host.ID,
		&host.HardwareUUID,
		&host.DisplayName,
		&host.Hostname,
		&host.ComputerName,
		&host.HardwareSerial,
		&host.HardwareModel,
		&host.Platform,
		&host.PlatformLike,
		&host.OSVersion,
		&host.OsqueryVersion,
		&host.OrbitVersion,
		&host.OrbitNodeKey,
		&host.OsqueryNodeKey,
		&host.CPUBrand,
		&host.CPULogicalCores,
		&host.CPUPhysicalCores,
		&host.PhysicalMemory,
		&host.HardwareVendor,
		&host.KernelVersion,
		&host.EnrolledAt,
		&host.LastSeenAt,
		&host.DetailUpdatedAt,
		&host.CreatedAt,
		&host.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return host, nil
}

// List returns all active hosts ordered most-recently-seen first.
func (s *HostStore) List(ctx context.Context) ([]Host, error) {
	rows, err := s.db.Query(ctx, `
SELECT id, hardware_uuid, display_name, hostname, computer_name,
       hardware_serial, hardware_model, platform, platform_like,
       os_version, osquery_version, orbit_version,
       COALESCE(orbit_node_key, ''), COALESCE(osquery_node_key, ''),
       cpu_brand, cpu_logical_cores, cpu_physical_cores, physical_memory,
       hardware_vendor, kernel_version,
       enrolled_at, last_seen_at, detail_updated_at,
       created_at, updated_at
FROM hosts
WHERE deleted_at IS NULL
ORDER BY last_seen_at DESC NULLS LAST, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hosts := make([]Host, 0)
	for rows.Next() {
		var host Host
		if err := scanHost(rows, &host); err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}
	return hosts, rows.Err()
}

// GetByID returns a single active host by database ID.
func (s *HostStore) GetByID(ctx context.Context, id int64) (*Host, error) {
	host := &Host{}
	row := s.db.QueryRow(ctx, `
SELECT id, hardware_uuid, display_name, hostname, computer_name,
       hardware_serial, hardware_model, platform, platform_like,
       os_version, osquery_version, orbit_version,
       COALESCE(orbit_node_key, ''), COALESCE(osquery_node_key, ''),
       cpu_brand, cpu_logical_cores, cpu_physical_cores, physical_memory,
       hardware_vendor, kernel_version,
       enrolled_at, last_seen_at, detail_updated_at,
       created_at, updated_at
FROM hosts
WHERE id = $1 AND deleted_at IS NULL`, id)
	if err := scanHost(row, host); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return host, nil
}

// GetByOrbitNodeKey returns an active host and refreshes last_seen_at.
func (s *HostStore) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.getByNodeKey(ctx, "orbit_node_key", nodeKey)
}

// GetByOsqueryNodeKey returns an active host and refreshes last_seen_at.
func (s *HostStore) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.getByNodeKey(ctx, "osquery_node_key", nodeKey)
}

func (s *HostStore) getByNodeKey(ctx context.Context, column, nodeKey string) (*Host, error) {
	if strings.TrimSpace(nodeKey) == "" {
		return nil, ErrNotFound
	}

	host := &Host{}
	row := s.db.QueryRow(ctx, `
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE `+column+` = $1 AND deleted_at IS NULL
RETURNING id, hardware_uuid, display_name, hostname, computer_name,
          hardware_serial, hardware_model, platform, platform_like,
          os_version, osquery_version, orbit_version,
          COALESCE(orbit_node_key, ''), COALESCE(osquery_node_key, ''),
          cpu_brand, cpu_logical_cores, cpu_physical_cores, physical_memory,
          hardware_vendor, kernel_version,
          enrolled_at, last_seen_at, detail_updated_at,
          created_at, updated_at`, nodeKey)
	if err := scanHost(row, host); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return host, nil
}

// ApplyDetail updates the host fields reported by successful detail queries.
func (s *HostStore) ApplyDetail(ctx context.Context, hostID int64, update HostDetailUpdate) error {
	return s.db.Exec(ctx, `
UPDATE hosts
SET hostname            = COALESCE(NULLIF($2, ''), hostname),
    computer_name       = COALESCE(NULLIF($3, ''), computer_name),
    display_name        = COALESCE(NULLIF($3, ''), NULLIF($2, ''), display_name),
    hardware_serial     = COALESCE(NULLIF($4, ''), hardware_serial),
    hardware_model      = COALESCE(NULLIF($5, ''), hardware_model),
    os_version          = COALESCE(NULLIF($6, ''), os_version),
    platform            = COALESCE(NULLIF($7, ''), platform),
    platform_like       = COALESCE(NULLIF($8, ''), platform_like),
    osquery_version     = COALESCE(NULLIF($9, ''), osquery_version),
    cpu_brand           = COALESCE(NULLIF($10, ''), cpu_brand),
    cpu_logical_cores   = CASE WHEN $11::integer > 0 THEN $11::integer ELSE cpu_logical_cores END,
    cpu_physical_cores  = CASE WHEN $12::integer > 0 THEN $12::integer ELSE cpu_physical_cores END,
    physical_memory     = CASE WHEN $13::bigint > 0 THEN $13::bigint ELSE physical_memory END,
    hardware_vendor     = COALESCE(NULLIF($14, ''), hardware_vendor),
    kernel_version      = COALESCE(NULLIF($15, ''), kernel_version),
    updated_at          = now()
WHERE id = $1 AND deleted_at IS NULL`,
		hostID,
		update.Hostname,
		update.ComputerName,
		update.HardwareSerial,
		update.HardwareModel,
		update.OSVersion,
		update.Platform,
		update.PlatformLike,
		update.OsqueryVersion,
		update.CPUBrand,
		update.CPULogicalCores,
		update.CPUPhysicalCores,
		update.PhysicalMemory,
		update.HardwareVendor,
		update.KernelVersion,
	)
}

// MarkDetailFresh records that all built-in detail queries completed.
func (s *HostStore) MarkDetailFresh(ctx context.Context, hostID int64) error {
	return s.db.Exec(ctx, `
UPDATE hosts
SET detail_updated_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL`, hostID)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHost(row rowScanner, host *Host) error {
	return row.Scan(
		&host.ID,
		&host.HardwareUUID,
		&host.DisplayName,
		&host.Hostname,
		&host.ComputerName,
		&host.HardwareSerial,
		&host.HardwareModel,
		&host.Platform,
		&host.PlatformLike,
		&host.OSVersion,
		&host.OsqueryVersion,
		&host.OrbitVersion,
		&host.OrbitNodeKey,
		&host.OsqueryNodeKey,
		&host.CPUBrand,
		&host.CPULogicalCores,
		&host.CPUPhysicalCores,
		&host.PhysicalMemory,
		&host.HardwareVendor,
		&host.KernelVersion,
		&host.EnrolledAt,
		&host.LastSeenAt,
		&host.DetailUpdatedAt,
		&host.CreatedAt,
		&host.UpdatedAt,
	)
}

// HostIDString formats a database host ID for API responses.
func HostIDString(id int64) string {
	return strconv.FormatInt(id, 10)
}

// displayNameFor picks the most user-friendly identifier from enroll params.
func displayNameFor(p EnrollParams) string {
	if v := strings.TrimSpace(p.ComputerName); v != "" {
		return v
	}
	if v := strings.TrimSpace(p.Hostname); v != "" {
		return v
	}
	return strings.TrimSpace(p.HardwareUUID)
}
