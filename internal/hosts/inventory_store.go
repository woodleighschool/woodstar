package hosts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

func (s *Store) ApplyInventory(ctx context.Context, hostID int64, update InventoryUpdate) error {
	write := applyInventoryWrite{
		ID:                                hostID,
		Hostname:                          update.Hostname,
		ComputerName:                      update.ComputerName,
		HardwareSerial:                    update.Hardware.Serial,
		HardwareModelIdentifier:           update.Hardware.ModelIdentifier,
		OSName:                            update.OS.Name,
		OSVersion:                         update.OS.Version,
		OSBuild:                           update.OS.Build,
		OSPlatform:                        update.OS.Platform,
		OsqueryVersion:                    update.Agents.Osquery.Version,
		OrbitVersion:                      update.Agents.Orbit.Version,
		OrbitScriptsEnabled:               update.Agents.Orbit.ScriptsEnabled,
		CPUType:                           update.Hardware.CPU.Architecture,
		CPUSubtype:                        update.Hardware.CPU.Subtype,
		CPUBrand:                          update.Hardware.CPU.Brand,
		CPULogicalCores:                   update.Hardware.CPU.LogicalCores,
		CPUPhysicalCores:                  update.Hardware.CPU.PhysicalCores,
		MemoryBytes:                       update.Hardware.MemoryBytes,
		HardwareVendor:                    update.Hardware.Vendor,
		OSKernelVersion:                   update.OS.KernelVersion,
		LastRestartedAt:                   update.LastRestartedAt,
		BootVolumeAvailableBytes:          update.Storage.BootVolume.AvailableBytes,
		BootVolumeTotalBytes:              update.Storage.BootVolume.TotalBytes,
		PrimaryIP:                         update.Network.PrimaryIP,
		PrimaryMAC:                        update.Network.PrimaryMAC,
		OsqueryDistributedIntervalSeconds: update.Agents.Osquery.DistributedIntervalSeconds,
		OsqueryConfigRefreshSeconds:       update.Agents.Osquery.ConfigRefreshSeconds,
	}
	_, err := s.pool.Exec(ctx, `
UPDATE hosts
SET
	hostname = COALESCE(NULLIF(@hostname::text, ''), hostname),
	computer_name = COALESCE(NULLIF(@computer_name::text, ''), computer_name),
	display_name = COALESCE(NULLIF(@computer_name::text, ''), NULLIF(@hostname::text, ''), display_name),
	hardware_serial = COALESCE(NULLIF(@hardware_serial::text, ''), hardware_serial),
	hardware_model_identifier = COALESCE(NULLIF(@hardware_model_identifier::text, ''), hardware_model_identifier),
	os_name = COALESCE(NULLIF(@os_name::text, ''), os_name),
	os_version = COALESCE(NULLIF(@os_version::text, ''), os_version),
	os_build = COALESCE(NULLIF(@os_build::text, ''), os_build),
	os_platform = COALESCE(NULLIF(@os_platform::text, ''), os_platform),
	osquery_version = COALESCE(NULLIF(@osquery_version::text, ''), osquery_version),
	orbit_version = COALESCE(NULLIF(@orbit_version::text, ''), orbit_version),
	orbit_scripts_enabled = COALESCE(@orbit_scripts_enabled::boolean, orbit_scripts_enabled),
	cpu_type = COALESCE(NULLIF(@cpu_type::text, ''), cpu_type),
	cpu_subtype = COALESCE(NULLIF(@cpu_subtype::text, ''), cpu_subtype),
	cpu_brand = COALESCE(NULLIF(@cpu_brand::text, ''), cpu_brand),
	cpu_logical_cores = CASE WHEN @cpu_logical_cores::integer > 0 THEN @cpu_logical_cores::integer ELSE cpu_logical_cores END,
	cpu_physical_cores = CASE WHEN @cpu_physical_cores::integer > 0 THEN @cpu_physical_cores::integer ELSE cpu_physical_cores END,
	memory_bytes = CASE WHEN @memory_bytes::bigint > 0 THEN @memory_bytes::bigint ELSE memory_bytes END,
	hardware_vendor = COALESCE(NULLIF(@hardware_vendor::text, ''), hardware_vendor),
	os_kernel_version = COALESCE(NULLIF(@os_kernel_version::text, ''), os_kernel_version),
	last_restarted_at = COALESCE(@last_restarted_at::timestamptz, last_restarted_at),
	boot_volume_available_bytes = COALESCE(@boot_volume_available_bytes::bigint, boot_volume_available_bytes),
	boot_volume_total_bytes = COALESCE(@boot_volume_total_bytes::bigint, boot_volume_total_bytes),
	primary_ip = COALESCE(NULLIF(@primary_ip::text, '')::inet, primary_ip),
	primary_mac = COALESCE(NULLIF(@primary_mac::text, ''), primary_mac),
	osquery_distributed_interval_seconds = COALESCE(@osquery_distributed_interval_seconds::integer, osquery_distributed_interval_seconds),
	osquery_config_refresh_seconds = COALESCE(@osquery_config_refresh_seconds::integer, osquery_config_refresh_seconds),
	updated_at = now()
WHERE id = @id`, pgx.StructArgs(write))
	return hostIdentityError(err)
}

func (s *Store) MarkInventoryFresh(ctx context.Context, hostID int64, inventoryQueryHash string) error {
	_, err := s.pool.Exec(ctx, `
UPDATE hosts
SET
	inventory_updated_at = now(),
	inventory_query_hash = @inventory_query_hash,
	inventory_refresh_requested = FALSE,
	updated_at = now()
WHERE id = @id`,
		pgx.NamedArgs{
			"id":                   hostID,
			"inventory_query_hash": inventoryQueryHash,
		})
	return err
}

// RequestInventoryRefresh marks a host's full osquery detail inventory as due.
func (s *Store) RequestInventoryRefresh(ctx context.Context, hostID int64) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE hosts
SET inventory_refresh_requested = TRUE, updated_at = now()
WHERE id = $1`, hostID)
	if err != nil {
		return postgres.GetError(err)
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

type applyInventoryWrite struct {
	ID                                int64      `db:"id"`
	Hostname                          string     `db:"hostname"`
	ComputerName                      string     `db:"computer_name"`
	HardwareSerial                    string     `db:"hardware_serial"`
	HardwareModelIdentifier           string     `db:"hardware_model_identifier"`
	OSName                            string     `db:"os_name"`
	OSVersion                         string     `db:"os_version"`
	OSBuild                           string     `db:"os_build"`
	OSPlatform                        string     `db:"os_platform"`
	OsqueryVersion                    string     `db:"osquery_version"`
	OrbitVersion                      string     `db:"orbit_version"`
	OrbitScriptsEnabled               *bool      `db:"orbit_scripts_enabled"`
	CPUType                           string     `db:"cpu_type"`
	CPUSubtype                        string     `db:"cpu_subtype"`
	CPUBrand                          string     `db:"cpu_brand"`
	CPULogicalCores                   int32      `db:"cpu_logical_cores"`
	CPUPhysicalCores                  int32      `db:"cpu_physical_cores"`
	MemoryBytes                       int64      `db:"memory_bytes"`
	HardwareVendor                    string     `db:"hardware_vendor"`
	OSKernelVersion                   string     `db:"os_kernel_version"`
	LastRestartedAt                   *time.Time `db:"last_restarted_at"`
	BootVolumeAvailableBytes          *int64     `db:"boot_volume_available_bytes"`
	BootVolumeTotalBytes              *int64     `db:"boot_volume_total_bytes"`
	PrimaryIP                         string     `db:"primary_ip"`
	PrimaryMAC                        string     `db:"primary_mac"`
	OsqueryDistributedIntervalSeconds *int32     `db:"osquery_distributed_interval_seconds"`
	OsqueryConfigRefreshSeconds       *int32     `db:"osquery_config_refresh_seconds"`
}

// Separate statements give the snapshot a fresh view of the previous writer's
// committed rows after taking the host lock.
func (s *Store) reconcileHostInventory(ctx context.Context, hostID int64, query string, rows any) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT id FROM hosts WHERE id = $1 FOR NO KEY UPDATE`, hostID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, query, hostID, rows)
		return err
	})
}
