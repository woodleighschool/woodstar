package hosts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/labels"
)

// UpsertOnOrbitEnroll creates or refreshes a host from Orbit enroll.
func (s *Store) UpsertOnOrbitEnroll(ctx context.Context, update InventoryUpdate) (*Host, error) {
	write := orbitEnrollWrite{
		HardwareUUID:            update.Hardware.UUID,
		DisplayName:             inventoryDisplayName(update.Hardware.UUID, update.Hostname, update.ComputerName),
		Hostname:                update.Hostname,
		ComputerName:            update.ComputerName,
		HardwareSerial:          update.Hardware.Serial,
		HardwareModelIdentifier: update.Hardware.ModelIdentifier,
		OrbitNodeKey:            update.OrbitNodeKey,
	}
	return upsertOnEnroll(ctx, s.db, `
INSERT INTO hosts (
	hardware_uuid,
	display_name,
	hostname,
	computer_name,
	hardware_serial,
	hardware_model_identifier,
	orbit_node_key,
	enrollment_agent,
	enrolled_at,
	last_seen_at
)
VALUES (
	@hardware_uuid,
	@display_name,
	@hostname,
	@computer_name,
	@hardware_serial,
	@hardware_model_identifier,
	@orbit_node_key,
	'orbit',
	now(),
	now()
)
ON CONFLICT (hardware_uuid) DO UPDATE SET
	display_name = EXCLUDED.display_name,
	hostname = EXCLUDED.hostname,
	computer_name = EXCLUDED.computer_name,
	hardware_serial = EXCLUDED.hardware_serial,
	hardware_model_identifier = EXCLUDED.hardware_model_identifier,
	orbit_node_key = EXCLUDED.orbit_node_key,
	orbit_device_auth_token = '',
	enrollment_agent = EXCLUDED.enrollment_agent,
	enrolled_at = now(),
	last_seen_at = now(),
	updated_at = now()
RETURNING`+hostColumnsSQL(), write)
}

// UpsertOnOsqueryEnroll creates or refreshes a host from osquery enroll.
func (s *Store) UpsertOnOsqueryEnroll(ctx context.Context, update InventoryUpdate) (*Host, error) {
	write := osqueryEnrollWrite{
		HardwareUUID:            update.Hardware.UUID,
		DisplayName:             inventoryDisplayName(update.Hardware.UUID, update.Hostname, update.ComputerName),
		Hostname:                update.Hostname,
		ComputerName:            update.ComputerName,
		HardwareSerial:          update.Hardware.Serial,
		HardwareModelIdentifier: update.Hardware.ModelIdentifier,
		OSName:                  update.OS.Name,
		OSVersion:               update.OS.Version,
		OSBuild:                 update.OS.Build,
		OSPlatform:              update.OS.Platform,
		OsqueryVersion:          update.Agents.Osquery.Version,
		OsqueryNodeKey:          update.OsqueryNodeKey,
		OrbitVersion:            update.Agents.Orbit.Version,
		CPUType:                 update.Hardware.CPU.Architecture,
		CPUSubtype:              update.Hardware.CPU.Subtype,
		CPUBrand:                update.Hardware.CPU.Brand,
		CPULogicalCores:         update.Hardware.CPU.LogicalCores,
		CPUPhysicalCores:        update.Hardware.CPU.PhysicalCores,
		MemoryBytes:             update.Hardware.MemoryBytes,
		HardwareVendor:          update.Hardware.Vendor,
		OSKernelVersion:         update.OS.KernelVersion,
	}
	return upsertOnEnroll(ctx, s.db, `
INSERT INTO hosts (
	hardware_uuid,
	display_name,
	hostname,
	computer_name,
	hardware_serial,
	hardware_model_identifier,
	os_version,
	os_name,
	os_build,
	os_platform,
	osquery_version,
	orbit_version,
	osquery_node_key,
	enrollment_agent,
	cpu_type,
	cpu_subtype,
	cpu_brand,
	cpu_logical_cores,
	cpu_physical_cores,
	memory_bytes,
	hardware_vendor,
	os_kernel_version,
	last_seen_at,
	inventory_updated_at
)
VALUES (
	@hardware_uuid,
	@display_name,
	@hostname,
	@computer_name,
	@hardware_serial,
	@hardware_model_identifier,
	@os_version,
	@os_name,
	@os_build,
	@os_platform,
	@osquery_version,
	@orbit_version,
	@osquery_node_key,
	'osquery',
	@cpu_type,
	@cpu_subtype,
	@cpu_brand,
	@cpu_logical_cores,
	@cpu_physical_cores,
	@memory_bytes,
	@hardware_vendor,
	@os_kernel_version,
	now(),
	NULL
)
ON CONFLICT (hardware_uuid) DO UPDATE SET
	display_name = EXCLUDED.display_name,
	hostname = EXCLUDED.hostname,
	computer_name = EXCLUDED.computer_name,
	hardware_serial = EXCLUDED.hardware_serial,
	hardware_model_identifier = EXCLUDED.hardware_model_identifier,
	os_version = EXCLUDED.os_version,
	os_name = COALESCE(NULLIF(EXCLUDED.os_name, ''), hosts.os_name),
	os_build = COALESCE(NULLIF(EXCLUDED.os_build, ''), hosts.os_build),
	os_platform = COALESCE(NULLIF(EXCLUDED.os_platform, ''), hosts.os_platform),
	osquery_version = EXCLUDED.osquery_version,
	orbit_version = COALESCE(NULLIF(EXCLUDED.orbit_version, ''), hosts.orbit_version),
	osquery_node_key = EXCLUDED.osquery_node_key,
	enrollment_agent = EXCLUDED.enrollment_agent,
	cpu_type = EXCLUDED.cpu_type,
	cpu_subtype = EXCLUDED.cpu_subtype,
	cpu_brand = EXCLUDED.cpu_brand,
	cpu_logical_cores = EXCLUDED.cpu_logical_cores,
	cpu_physical_cores = EXCLUDED.cpu_physical_cores,
	memory_bytes = EXCLUDED.memory_bytes,
	hardware_vendor = EXCLUDED.hardware_vendor,
	os_kernel_version = EXCLUDED.os_kernel_version,
	inventory_updated_at = NULL,
	inventory_query_hash = '',
	last_seen_at = now(),
	updated_at = now()
RETURNING`+hostColumnsSQL(), write)
}

func upsertOnEnroll[W any](ctx context.Context, db *database.DB, sql string, write W) (*Host, error) {
	now := time.Now()
	var host Host
	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sql, pgx.StructArgs(write))
		if err != nil {
			return err
		}
		row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[hostRow])
		if err != nil {
			return err
		}
		host = hostFromRow(row, now)
		_, err = tx.Exec(ctx, `
INSERT INTO label_membership (label_id, host_id)
SELECT id, @host_id
FROM labels
WHERE builtin_key = @builtin_key::text AND label_type = 'builtin' AND label_membership_type = 'manual'
ON CONFLICT (label_id, host_id) DO NOTHING`,
			pgx.NamedArgs{
				"host_id":     host.ID,
				"builtin_key": string(labels.BuiltinKeyAllHosts),
			})
		return err
	})
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// inventoryDisplayName persists the canonical host label exposed by the API.
func inventoryDisplayName(hardwareUUID, hostname, computerName string) string {
	if computerName != "" {
		return computerName
	}
	if hostname != "" {
		return hostname
	}
	return hardwareUUID
}

type orbitEnrollWrite struct {
	HardwareUUID            string `db:"hardware_uuid"`
	DisplayName             string `db:"display_name"`
	Hostname                string `db:"hostname"`
	ComputerName            string `db:"computer_name"`
	HardwareSerial          string `db:"hardware_serial"`
	HardwareModelIdentifier string `db:"hardware_model_identifier"`
	OrbitNodeKey            string `db:"orbit_node_key"`
}

type osqueryEnrollWrite struct {
	HardwareUUID            string `db:"hardware_uuid"`
	DisplayName             string `db:"display_name"`
	Hostname                string `db:"hostname"`
	ComputerName            string `db:"computer_name"`
	HardwareSerial          string `db:"hardware_serial"`
	HardwareModelIdentifier string `db:"hardware_model_identifier"`
	OSName                  string `db:"os_name"`
	OSVersion               string `db:"os_version"`
	OSBuild                 string `db:"os_build"`
	OSPlatform              string `db:"os_platform"`
	OsqueryVersion          string `db:"osquery_version"`
	OsqueryNodeKey          string `db:"osquery_node_key"`
	OrbitVersion            string `db:"orbit_version"`
	CPUType                 string `db:"cpu_type"`
	CPUSubtype              string `db:"cpu_subtype"`
	CPUBrand                string `db:"cpu_brand"`
	CPULogicalCores         int32  `db:"cpu_logical_cores"`
	CPUPhysicalCores        int32  `db:"cpu_physical_cores"`
	MemoryBytes             int64  `db:"memory_bytes"`
	HardwareVendor          string `db:"hardware_vendor"`
	OSKernelVersion         string `db:"os_kernel_version"`
}
