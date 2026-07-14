package hosts

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
)

// Store persists hosts.
type Store struct {
	db     *database.DB
	labels hostLabelReader
}

type hostLabelReader interface {
	ListForHost(context.Context, int64) ([]labels.Label, error)
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, labels: labels.NewStore(db)}
}

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
//
//nolint:funlen // Keep the osquery enrollment mutation with the store method that owns it.
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

func (s *Store) List(ctx context.Context, params HostListParams) ([]Host, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	where, args := hostListWhere(params)
	listQuery := hostListQuery(params, where, args)
	rows, count, err := dbutil.ListWithCount[hostRow](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	hosts := make([]Host, len(rows))
	for i, row := range rows {
		hosts[i] = hostFromRow(row, now)
	}
	if err := s.attachPrimaryUser(ctx, hosts); err != nil {
		return nil, 0, err
	}
	return hosts, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Host, error) {
	row, err := dbutil.GetOne[hostRow](ctx, s.db.Pool(), hostSelectSQL()+"\nWHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	host := hostFromRow(row, time.Now())
	return &host, nil
}

// GetByHardwareSerial returns the existing host with serial.
func (s *Store) GetByHardwareSerial(ctx context.Context, serial string) (*Host, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil, dbutil.ErrNotFound
	}
	rows, err := s.db.Pool().Query(ctx, hostSelectSQL()+`
WHERE hardware_serial = $1 AND hardware_serial <> ''
ORDER BY updated_at DESC, id DESC
LIMIT 2`, serial)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostRow])
	if err != nil {
		return nil, err
	}
	switch len(records) {
	case 0:
		return nil, dbutil.ErrNotFound
	case 1:
		host := hostFromRow(records[0], time.Now())
		return &host, nil
	default:
		return nil, fmt.Errorf("multiple hosts have hardware serial %q", serial)
	}
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, `DELETE FROM hosts WHERE id = $1`, id)
	if err != nil {
		return dbutil.GetError(err)
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteMany removes hosts. Missing IDs are fine.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.db.Pool().Query(ctx, `DELETE FROM hosts WHERE id = ANY($1::bigint[]) RETURNING id`, ids)
	if err != nil {
		return 0, err
	}
	deleted, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	return len(deleted), nil
}

func (s *Store) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.touchByNodeKey(ctx, hostTouchSQL("orbit_node_key"), nodeKey)
}

func (s *Store) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.touchByNodeKey(ctx, hostTouchSQL("osquery_node_key"), nodeKey)
}

// SetOrbitDeviceAuthToken replaces the machine token for an Orbit node key.
func (s *Store) SetOrbitDeviceAuthToken(ctx context.Context, nodeKey, token string) error {
	tag, err := s.db.Pool().Exec(ctx, `
UPDATE hosts
SET
    orbit_device_auth_token = $2,
    last_seen_at = now(),
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
func (s *Store) ValidateOrbitDeviceAuthToken(ctx context.Context, token string) error {
	var hostID int64
	err := s.db.Pool().QueryRow(ctx, `
WITH touched AS (
    UPDATE hosts
    SET last_seen_at = now()
    WHERE orbit_device_auth_token = $1
      AND orbit_device_auth_token <> ''
      AND (last_seen_at IS NULL OR last_seen_at < now() - interval '1 minute')
    RETURNING id
)
SELECT id FROM touched
UNION ALL
SELECT id
FROM hosts
WHERE orbit_device_auth_token = $1
  AND orbit_device_auth_token <> ''
  AND NOT EXISTS (SELECT 1 FROM touched)
LIMIT 1`, token).Scan(&hostID)
	return dbutil.GetError(err)
}

func hostTouchSQL(nodeKeyColumn string) string {
	return `
WITH touched AS (
    UPDATE hosts
    SET last_seen_at = now()
    WHERE ` + nodeKeyColumn + ` = $1
      AND ` + nodeKeyColumn + ` <> ''
      AND (last_seen_at IS NULL OR last_seen_at < now() - interval '1 minute')
    RETURNING` + hostColumnsSQL() + `
)
SELECT` + hostColumnsSQL() + `
FROM touched
UNION ALL
SELECT` + hostColumnsSQL() + `
FROM hosts
WHERE ` + nodeKeyColumn + ` = $1
  AND ` + nodeKeyColumn + ` <> ''
  AND NOT EXISTS (SELECT 1 FROM touched)
LIMIT 1`
}

func (s *Store) touchByNodeKey(ctx context.Context, sql, nodeKey string) (*Host, error) {
	row, err := dbutil.GetOne[hostRow](ctx, s.db.Pool(), sql, nodeKey)
	if err != nil {
		return nil, err
	}
	host := hostFromRow(row, time.Now())
	return &host, nil
}

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
		CPUType:                           update.Hardware.CPU.Architecture,
		CPUSubtype:                        update.Hardware.CPU.Subtype,
		CPUBrand:                          update.Hardware.CPU.Brand,
		CPULogicalCores:                   update.Hardware.CPU.LogicalCores,
		CPUPhysicalCores:                  update.Hardware.CPU.PhysicalCores,
		MemoryBytes:                       update.Hardware.MemoryBytes,
		HardwareVendor:                    update.Hardware.Vendor,
		OSKernelVersion:                   update.OS.KernelVersion,
		LastRestartedAt:                   update.Timestamps.LastRestartedAt,
		BootVolumeAvailableBytes:          update.Storage.BootVolume.AvailableBytes,
		BootVolumeTotalBytes:              update.Storage.BootVolume.TotalBytes,
		LastRemoteIP:                      update.Network.LastRemoteIP,
		PrimaryIP:                         update.Network.PrimaryIP,
		PrimaryMAC:                        update.Network.PrimaryMAC,
		OsqueryDistributedIntervalSeconds: update.Agents.Osquery.DistributedIntervalSeconds,
		OsqueryConfigRefreshSeconds:       update.Agents.Osquery.ConfigRefreshSeconds,
	}
	_, err := s.db.Pool().Exec(ctx, `
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
	last_remote_ip = COALESCE(NULLIF(@last_remote_ip::text, '')::inet, last_remote_ip),
	primary_ip = COALESCE(NULLIF(@primary_ip::text, '')::inet, primary_ip),
	primary_mac = COALESCE(NULLIF(@primary_mac::text, ''), primary_mac),
	osquery_distributed_interval_seconds = COALESCE(@osquery_distributed_interval_seconds::integer, osquery_distributed_interval_seconds),
	osquery_config_refresh_seconds = COALESCE(@osquery_config_refresh_seconds::integer, osquery_config_refresh_seconds),
	updated_at = now()
WHERE id = @id`, pgx.StructArgs(write))
	return err
}

func (s *Store) ReplaceUsers(ctx context.Context, hostID int64, users []HostUser) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM host_users WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		for _, user := range users {
			if user.UID == "" || user.Username == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `
INSERT INTO host_users (
	host_id,
	uid,
	username,
	type,
	description,
	directory,
	shell
)
VALUES (
	@host_id,
	@uid,
	@username,
	@type,
	@description,
	@directory,
	@shell
)
ON CONFLICT (host_id, uid, username) DO UPDATE SET
	type = EXCLUDED.type,
	description = EXCLUDED.description,
	directory = EXCLUDED.directory,
	shell = EXCLUDED.shell`, pgx.StructArgs(newHostUserWrite(hostID, user))); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ReplaceBatteries(ctx context.Context, hostID int64, batteries []HostBattery) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM host_batteries WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		for _, battery := range batteries {
			if battery.SerialNumber == "" {
				continue
			}
			if _, err := tx.Exec(
				ctx,
				`
INSERT INTO host_batteries (
	host_id,
	serial_number,
	manufacturer,
	model,
	chemistry,
	cycle_count,
	health,
	designed_capacity,
	max_capacity,
	current_capacity,
	percent_remaining
)
VALUES (
	@host_id,
	@serial_number,
	@manufacturer,
	@model,
	@chemistry,
	@cycle_count,
	@health,
	@designed_capacity,
	@max_capacity,
	@current_capacity,
	@percent_remaining
)
ON CONFLICT (host_id, serial_number) DO UPDATE SET
	manufacturer = EXCLUDED.manufacturer,
	model = EXCLUDED.model,
	chemistry = EXCLUDED.chemistry,
	cycle_count = EXCLUDED.cycle_count,
	health = EXCLUDED.health,
	designed_capacity = EXCLUDED.designed_capacity,
	max_capacity = EXCLUDED.max_capacity,
	current_capacity = EXCLUDED.current_capacity,
	percent_remaining = EXCLUDED.percent_remaining`,
				pgx.StructArgs(newHostBatteryWrite(hostID, battery)),
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ReplaceCertificates(ctx context.Context, hostID int64, certificates []HostCertificate) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM host_certificates WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		for _, certificate := range certificates {
			if certificate.SHA1 == "" {
				continue
			}
			if _, err := tx.Exec(
				ctx,
				`
INSERT INTO host_certificates (
	host_id,
	sha1,
	common_name,
	subject_country,
	subject_organization,
	subject_organizational_unit,
	subject_common_name,
	issuer_country,
	issuer_organization,
	issuer_organizational_unit,
	issuer_common_name,
	key_algorithm,
	key_strength,
	key_usage,
	signing_algorithm,
	not_valid_after,
	not_valid_before,
	serial,
	certificate_authority,
	source,
	username,
	path
)
VALUES (
	@host_id,
	@sha1,
	@common_name,
	@subject_country,
	@subject_organization,
	@subject_organizational_unit,
	@subject_common_name,
	@issuer_country,
	@issuer_organization,
	@issuer_organizational_unit,
	@issuer_common_name,
	@key_algorithm,
	@key_strength,
	@key_usage,
	@signing_algorithm,
	@not_valid_after,
	@not_valid_before,
	@serial,
	@certificate_authority,
	@source,
	@username,
	@path
)
ON CONFLICT (host_id, sha1, source, username) DO UPDATE SET
	common_name = EXCLUDED.common_name,
	subject_country = EXCLUDED.subject_country,
	subject_organization = EXCLUDED.subject_organization,
	subject_organizational_unit = EXCLUDED.subject_organizational_unit,
	subject_common_name = EXCLUDED.subject_common_name,
	issuer_country = EXCLUDED.issuer_country,
	issuer_organization = EXCLUDED.issuer_organization,
	issuer_organizational_unit = EXCLUDED.issuer_organizational_unit,
	issuer_common_name = EXCLUDED.issuer_common_name,
	key_algorithm = EXCLUDED.key_algorithm,
	key_strength = EXCLUDED.key_strength,
	key_usage = EXCLUDED.key_usage,
	signing_algorithm = EXCLUDED.signing_algorithm,
	not_valid_after = EXCLUDED.not_valid_after,
	not_valid_before = EXCLUDED.not_valid_before,
	serial = EXCLUDED.serial,
	certificate_authority = EXCLUDED.certificate_authority,
	path = EXCLUDED.path`,
				pgx.StructArgs(newHostCertificateWrite(hostID, certificate)),
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListUsers(ctx context.Context, hostID int64) ([]HostUser, error) {
	rows, err := s.db.Pool().Query(ctx, `
SELECT id, host_id, uid, username, type, description, directory, shell
FROM host_users
WHERE host_id = $1
ORDER BY username, uid, id`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[HostUser])
}

func (s *Store) ListBatteries(ctx context.Context, hostID int64) ([]HostBattery, error) {
	rows, err := s.db.Pool().Query(ctx, `
SELECT
	id, host_id, serial_number, manufacturer, model, chemistry, cycle_count,
	health, designed_capacity, max_capacity, current_capacity, percent_remaining
FROM host_batteries
WHERE host_id = $1
ORDER BY serial_number, id`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[HostBattery])
}

func (s *Store) ListCertificates(ctx context.Context, hostID int64) ([]HostCertificate, error) {
	rows, err := s.db.Pool().Query(ctx, `
SELECT
	id, host_id, sha1, common_name,
	subject_country, subject_organization, subject_organizational_unit, subject_common_name,
	issuer_country, issuer_organization, issuer_organizational_unit, issuer_common_name,
	key_algorithm, key_strength, key_usage, signing_algorithm,
	not_valid_after, not_valid_before, serial, certificate_authority,
	source, username, path
FROM host_certificates
WHERE host_id = $1
ORDER BY common_name, sha1, id`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostCertificateRow])
	if err != nil {
		return nil, err
	}
	certificates := make([]HostCertificate, len(records))
	for i, record := range records {
		certificates[i] = hostCertificateFromRow(record)
	}
	return certificates, nil
}

func (s *Store) MarkInventoryFresh(ctx context.Context, hostID int64, inventoryQueryHash string) error {
	_, err := s.db.Pool().Exec(ctx, `
UPDATE hosts
SET inventory_updated_at = now(), inventory_query_hash = @inventory_query_hash, updated_at = now()
WHERE id = @id`,
		pgx.NamedArgs{
			"id":                   hostID,
			"inventory_query_hash": inventoryQueryHash,
		})
	return err
}

func (s *Store) attachPrimaryUser(ctx context.Context, hosts []Host) error {
	if len(hosts) == 0 {
		return nil
	}
	hostIDs := make([]int64, len(hosts))
	for i := range hosts {
		hostIDs[i] = hosts[i].ID
	}
	primaryUsers, err := s.loadPrimaryUser(ctx, hostIDs)
	if err != nil {
		return err
	}
	for i := range hosts {
		primaryUser := primaryUsers[hosts[i].ID]
		hosts[i].PrimaryUser = primaryUser.Primary
		hosts[i].PrimaryUserSources = primaryUser.Sources
	}
	return nil
}

func hostListQuery(params HostListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: hostSelectSQL(),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":                        {SQL: "lower(display_name)"},
			"hardware.serial":                     {SQL: "lower(hardware_serial)"},
			"hardware.model_identifier":           {SQL: "lower(hardware_model_identifier)"},
			"hardware.uuid":                       {SQL: "hardware_uuid"},
			"os.version":                          {SQL: "lower(os_version)"},
			"agents.osquery.version":              {SQL: "lower(osquery_version)"},
			"timestamps.last_seen_at":             {SQL: "last_seen_at", NullOrder: dbutil.NullsLast},
			"timestamps.last_restarted_at":        {SQL: "last_restarted_at", NullOrder: dbutil.NullsLast},
			"storage.boot_volume.available_bytes": {SQL: "boot_volume_available_bytes", NullOrder: dbutil.NullsLast},
			"hardware.memory_bytes":               {SQL: "memory_bytes"},
			"network.primary_ip":                  {SQL: "primary_ip", NullOrder: dbutil.NullsLast},
			"network.last_remote_ip":              {SQL: "last_remote_ip", NullOrder: dbutil.NullsLast},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}
}

func hostListWhere(params HostListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			display_name ILIKE ` + search + `
			OR hostname ILIKE ` + search + `
				OR computer_name ILIKE ` + search + `
				OR hardware_serial ILIKE ` + search + `
				OR hardware_uuid ILIKE ` + search + `
				OR hardware_model_identifier ILIKE ` + search + `
				OR os_version ILIKE ` + search + `
				OR EXISTS (
					SELECT 1 FROM host_primary_user_sources s
					WHERE s.host_id = hosts.id AND s.email ILIKE ` + search + `
				)
			)`)
	}
	if len(params.IDs) > 0 {
		where.Addf("id = ANY(%s::bigint[])", params.IDs)
	}
	switch params.Status {
	case "":
	case HostStatusOnline:
		where.Add("last_seen_at >= now() - interval '5 minutes'")
	case HostStatusOffline:
		where.Add("(last_seen_at IS NULL OR last_seen_at < now() - interval '5 minutes')")
	}
	if params.LabelID != 0 {
		labelID := where.Arg(params.LabelID)
		where.Add(`EXISTS (
			SELECT 1 FROM label_membership lm
			WHERE lm.host_id = hosts.id AND lm.label_id = ` + labelID + `::bigint
		)`)
	}
	if params.SoftwareID != 0 {
		softwareID := where.Arg(params.SoftwareID)
		where.Add(`EXISTS (
			SELECT 1 FROM host_software hs
			WHERE hs.host_id = hosts.id AND hs.software_id = ` + softwareID + `::bigint
		)`)
	}
	if params.SoftwareTitleID != 0 {
		softwareTitleID := where.Arg(params.SoftwareTitleID)
		where.Add(`EXISTS (
			SELECT 1
			FROM host_software hs
			JOIN software s ON s.id = hs.software_id
			WHERE hs.host_id = hosts.id AND s.title_id = ` + softwareTitleID + `::bigint
		)`)
	}
	return where.Build()
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

func statusFromLastSeen(lastSeen *time.Time, now time.Time) HostStatus {
	if lastSeen == nil || lastSeen.Before(now.Add(-hostOnlineWindow)) {
		return HostStatusOffline
	}
	return HostStatusOnline
}

func (s *Store) loadPrimaryUser(ctx context.Context, hostIDs []int64) (map[int64]hostPrimaryUser, error) {
	primaryUsers := make(map[int64]hostPrimaryUser, len(hostIDs))
	for _, hostID := range hostIDs {
		primaryUsers[hostID] = hostPrimaryUser{Sources: []HostPrimaryUserSource{}}
	}
	if len(hostIDs) == 0 {
		return primaryUsers, nil
	}

	sourceRows, err := s.db.Pool().Query(ctx, listHostPrimaryUserSourcesForHostsSQL, hostIDs)
	if err != nil {
		return nil, err
	}
	sources, err := pgx.CollectRows(sourceRows, pgx.RowToStructByName[hostPrimaryUserSourceRow])
	if err != nil {
		return nil, err
	}
	grouped := groupHostPrimaryUserSources(sources)

	primaryRows, err := s.db.Pool().Query(ctx, listHostPrimaryUsersSQL, hostIDs)
	if err != nil {
		return nil, err
	}
	primaries, err := pgx.CollectRows(primaryRows, pgx.RowToStructByName[hostPrimaryUserRow])
	if err != nil {
		return nil, err
	}
	for _, row := range primaries {
		primaryUser := primaryUsers[row.HostID]
		primaryUser.Primary = &HostPrimaryUser{
			Email:      row.Email,
			Username:   row.Username,
			Name:       row.Name,
			Department: row.Department,
			Groups:     row.Groups,
			Source:     PrimaryUserSource(row.Source),
		}
		primaryUsers[row.HostID] = primaryUser
	}
	for hostID, sources := range grouped {
		primaryUser := primaryUsers[hostID]
		primaryUser.Sources = sources
		primaryUsers[hostID] = primaryUser
	}
	return primaryUsers, nil
}

// hostRow is the canonical scan target for the hosts projection.
type hostRow struct {
	ID                                int64       `db:"id"`
	HardwareUUID                      string      `db:"hardware_uuid"`
	DisplayName                       string      `db:"display_name"`
	Hostname                          string      `db:"hostname"`
	ComputerName                      string      `db:"computer_name"`
	HardwareSerial                    string      `db:"hardware_serial"`
	HardwareModelIdentifier           string      `db:"hardware_model_identifier"`
	HardwareVendor                    string      `db:"hardware_vendor"`
	OSName                            string      `db:"os_name"`
	OSVersion                         string      `db:"os_version"`
	OSBuild                           string      `db:"os_build"`
	OSPlatform                        string      `db:"os_platform"`
	OsqueryVersion                    string      `db:"osquery_version"`
	OrbitVersion                      string      `db:"orbit_version"`
	OrbitNodeKey                      string      `db:"orbit_node_key"`
	OsqueryNodeKey                    string      `db:"osquery_node_key"`
	EnrollmentAgent                   string      `db:"enrollment_agent"`
	CPUType                           string      `db:"cpu_type"`
	CPUSubtype                        string      `db:"cpu_subtype"`
	CPUBrand                          string      `db:"cpu_brand"`
	CPULogicalCores                   int32       `db:"cpu_logical_cores"`
	CPUPhysicalCores                  int32       `db:"cpu_physical_cores"`
	MemoryBytes                       int64       `db:"memory_bytes"`
	OSKernelVersion                   string      `db:"os_kernel_version"`
	LastRestartedAt                   *time.Time  `db:"last_restarted_at"`
	BootVolumeAvailableBytes          *int64      `db:"boot_volume_available_bytes"`
	BootVolumeTotalBytes              *int64      `db:"boot_volume_total_bytes"`
	LastRemoteIP                      *netip.Addr `db:"last_remote_ip"`
	PrimaryIP                         *netip.Addr `db:"primary_ip"`
	PrimaryMAC                        string      `db:"primary_mac"`
	OsqueryDistributedIntervalSeconds *int32      `db:"osquery_distributed_interval_seconds"`
	OsqueryConfigRefreshSeconds       *int32      `db:"osquery_config_refresh_seconds"`
	InventoryQueryHash                string      `db:"inventory_query_hash"`
	EnrolledAt                        *time.Time  `db:"enrolled_at"`
	LastSeenAt                        *time.Time  `db:"last_seen_at"`
	InventoryUpdatedAt                *time.Time  `db:"inventory_updated_at"`
	CreatedAt                         time.Time   `db:"created_at"`
	UpdatedAt                         time.Time   `db:"updated_at"`
}

func hostColumnsSQL() string {
	return `
	id,
	hardware_uuid,
	display_name,
	hostname,
	computer_name,
	hardware_serial,
	hardware_model_identifier,
	hardware_vendor,
	os_name,
	os_version,
	os_build,
	os_platform,
	osquery_version,
	orbit_version,
	orbit_node_key,
	osquery_node_key,
	enrollment_agent,
	cpu_type,
	cpu_subtype,
	cpu_brand,
	cpu_logical_cores,
	cpu_physical_cores,
	memory_bytes,
	os_kernel_version,
	last_restarted_at,
	boot_volume_available_bytes,
	boot_volume_total_bytes,
	last_remote_ip,
	primary_ip,
	primary_mac,
	osquery_distributed_interval_seconds,
	osquery_config_refresh_seconds,
	inventory_query_hash,
	enrolled_at,
	last_seen_at,
	inventory_updated_at,
	created_at,
	updated_at`
}

func hostSelectSQL() string {
	return `SELECT` + hostColumnsSQL() + `
FROM hosts`
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
	LastRemoteIP                      string     `db:"last_remote_ip"`
	PrimaryIP                         string     `db:"primary_ip"`
	PrimaryMAC                        string     `db:"primary_mac"`
	OsqueryDistributedIntervalSeconds *int32     `db:"osquery_distributed_interval_seconds"`
	OsqueryConfigRefreshSeconds       *int32     `db:"osquery_config_refresh_seconds"`
}

type hostUserWrite struct {
	HostID      int64  `db:"host_id"`
	UID         string `db:"uid"`
	Username    string `db:"username"`
	Type        string `db:"type"`
	Description string `db:"description"`
	Directory   string `db:"directory"`
	Shell       string `db:"shell"`
}

func newHostUserWrite(hostID int64, user HostUser) hostUserWrite {
	return hostUserWrite{
		HostID:      hostID,
		UID:         user.UID,
		Username:    user.Username,
		Type:        user.Type,
		Description: user.Description,
		Directory:   user.Directory,
		Shell:       user.Shell,
	}
}

type hostBatteryWrite struct {
	HostID           int64    `db:"host_id"`
	SerialNumber     string   `db:"serial_number"`
	Manufacturer     string   `db:"manufacturer"`
	Model            string   `db:"model"`
	Chemistry        string   `db:"chemistry"`
	CycleCount       *int32   `db:"cycle_count"`
	Health           string   `db:"health"`
	DesignedCapacity *int32   `db:"designed_capacity"`
	MaxCapacity      *int32   `db:"max_capacity"`
	CurrentCapacity  *int32   `db:"current_capacity"`
	PercentRemaining *float64 `db:"percent_remaining"`
}

func newHostBatteryWrite(hostID int64, battery HostBattery) hostBatteryWrite {
	return hostBatteryWrite{
		HostID:           hostID,
		SerialNumber:     battery.SerialNumber,
		Manufacturer:     battery.Manufacturer,
		Model:            battery.Model,
		Chemistry:        battery.Chemistry,
		CycleCount:       battery.CycleCount,
		Health:           battery.Health,
		DesignedCapacity: battery.DesignedCapacity,
		MaxCapacity:      battery.MaxCapacity,
		CurrentCapacity:  battery.CurrentCapacity,
		PercentRemaining: battery.PercentRemaining,
	}
}

type hostCertificateWrite struct {
	HostID                    int64      `db:"host_id"`
	SHA1                      string     `db:"sha1"`
	CommonName                string     `db:"common_name"`
	SubjectCountry            string     `db:"subject_country"`
	SubjectOrganization       string     `db:"subject_organization"`
	SubjectOrganizationalUnit string     `db:"subject_organizational_unit"`
	SubjectCommonName         string     `db:"subject_common_name"`
	IssuerCountry             string     `db:"issuer_country"`
	IssuerOrganization        string     `db:"issuer_organization"`
	IssuerOrganizationalUnit  string     `db:"issuer_organizational_unit"`
	IssuerCommonName          string     `db:"issuer_common_name"`
	KeyAlgorithm              string     `db:"key_algorithm"`
	KeyStrength               *int32     `db:"key_strength"`
	KeyUsage                  string     `db:"key_usage"`
	SigningAlgorithm          string     `db:"signing_algorithm"`
	NotValidAfter             *time.Time `db:"not_valid_after"`
	NotValidBefore            *time.Time `db:"not_valid_before"`
	Serial                    string     `db:"serial"`
	CertificateAuthority      bool       `db:"certificate_authority"`
	Source                    string     `db:"source"`
	Username                  string     `db:"username"`
	Path                      string     `db:"path"`
}

func newHostCertificateWrite(hostID int64, certificate HostCertificate) hostCertificateWrite {
	return hostCertificateWrite{
		HostID:                    hostID,
		SHA1:                      certificate.SHA1,
		CommonName:                certificate.CommonName,
		SubjectCountry:            certificate.Subject.Country,
		SubjectOrganization:       certificate.Subject.Organization,
		SubjectOrganizationalUnit: certificate.Subject.OrganizationalUnit,
		SubjectCommonName:         certificate.Subject.CommonName,
		IssuerCountry:             certificate.Issuer.Country,
		IssuerOrganization:        certificate.Issuer.Organization,
		IssuerOrganizationalUnit:  certificate.Issuer.OrganizationalUnit,
		IssuerCommonName:          certificate.Issuer.CommonName,
		KeyAlgorithm:              certificate.KeyAlgorithm,
		KeyStrength:               certificate.KeyStrength,
		KeyUsage:                  certificate.KeyUsage,
		SigningAlgorithm:          certificate.SigningAlgorithm,
		NotValidAfter:             certificate.NotValidAfter,
		NotValidBefore:            certificate.NotValidBefore,
		Serial:                    certificate.Serial,
		CertificateAuthority:      certificate.CertificateAuthority,
		Source:                    certificate.Source,
		Username:                  certificate.Username,
		Path:                      certificate.Path,
	}
}

func hostFromRow(row hostRow, now time.Time) Host {
	return Host{
		ID:           row.ID,
		DisplayName:  row.DisplayName,
		Status:       statusFromLastSeen(row.LastSeenAt, now),
		Hostname:     row.Hostname,
		ComputerName: row.ComputerName,
		Enrollment: HostEnrollment{
			Agent:      row.EnrollmentAgent,
			EnrolledAt: row.EnrolledAt,
		},
		Hardware: HostHardware{
			UUID:            row.HardwareUUID,
			Serial:          row.HardwareSerial,
			Vendor:          row.HardwareVendor,
			ModelIdentifier: row.HardwareModelIdentifier,
			MemoryBytes:     row.MemoryBytes,
			CPU: HostCPU{
				Architecture:  row.CPUType,
				Subtype:       row.CPUSubtype,
				Brand:         row.CPUBrand,
				LogicalCores:  row.CPULogicalCores,
				PhysicalCores: row.CPUPhysicalCores,
			},
		},
		OS: HostOS{
			Platform:      row.OSPlatform,
			Name:          row.OSName,
			Version:       row.OSVersion,
			Build:         row.OSBuild,
			KernelVersion: row.OSKernelVersion,
		},
		Storage: HostStorage{
			BootVolume: HostBootVolume{
				AvailableBytes: row.BootVolumeAvailableBytes,
				TotalBytes:     row.BootVolumeTotalBytes,
			},
		},
		Network: HostNetwork{
			PrimaryIP:    row.PrimaryIP,
			PrimaryMAC:   row.PrimaryMAC,
			LastRemoteIP: row.LastRemoteIP,
		},
		Agents: HostAgents{
			Osquery: HostOsqueryAgent{
				Version:                    row.OsqueryVersion,
				DistributedIntervalSeconds: row.OsqueryDistributedIntervalSeconds,
				ConfigRefreshSeconds:       row.OsqueryConfigRefreshSeconds,
			},
			Orbit: HostOrbitAgent{Version: row.OrbitVersion},
		},
		PrimaryUserSources: []HostPrimaryUserSource{},
		Timestamps: HostTimestamps{
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
			LastSeenAt:         row.LastSeenAt,
			InventoryUpdatedAt: row.InventoryUpdatedAt,
			LastRestartedAt:    row.LastRestartedAt,
		},
		OrbitNodeKey:       row.OrbitNodeKey,
		OsqueryNodeKey:     row.OsqueryNodeKey,
		InventoryQueryHash: row.InventoryQueryHash,
	}
}

type hostCertificateRow struct {
	ID                        int64      `db:"id"`
	HostID                    int64      `db:"host_id"`
	SHA1                      string     `db:"sha1"`
	CommonName                string     `db:"common_name"`
	SubjectCountry            string     `db:"subject_country"`
	SubjectOrganization       string     `db:"subject_organization"`
	SubjectOrganizationalUnit string     `db:"subject_organizational_unit"`
	SubjectCommonName         string     `db:"subject_common_name"`
	IssuerCountry             string     `db:"issuer_country"`
	IssuerOrganization        string     `db:"issuer_organization"`
	IssuerOrganizationalUnit  string     `db:"issuer_organizational_unit"`
	IssuerCommonName          string     `db:"issuer_common_name"`
	KeyAlgorithm              string     `db:"key_algorithm"`
	KeyStrength               *int32     `db:"key_strength"`
	KeyUsage                  string     `db:"key_usage"`
	SigningAlgorithm          string     `db:"signing_algorithm"`
	NotValidAfter             *time.Time `db:"not_valid_after"`
	NotValidBefore            *time.Time `db:"not_valid_before"`
	Serial                    string     `db:"serial"`
	CertificateAuthority      bool       `db:"certificate_authority"`
	Source                    string     `db:"source"`
	Username                  string     `db:"username"`
	Path                      string     `db:"path"`
}

func hostCertificateFromRow(row hostCertificateRow) HostCertificate {
	return HostCertificate{
		ID:         row.ID,
		HostID:     row.HostID,
		SHA1:       row.SHA1,
		CommonName: row.CommonName,
		Subject: CertificateName{
			Country:            row.SubjectCountry,
			Organization:       row.SubjectOrganization,
			OrganizationalUnit: row.SubjectOrganizationalUnit,
			CommonName:         row.SubjectCommonName,
		},
		Issuer: CertificateName{
			Country:            row.IssuerCountry,
			Organization:       row.IssuerOrganization,
			OrganizationalUnit: row.IssuerOrganizationalUnit,
			CommonName:         row.IssuerCommonName,
		},
		KeyAlgorithm:         row.KeyAlgorithm,
		KeyStrength:          row.KeyStrength,
		KeyUsage:             row.KeyUsage,
		SigningAlgorithm:     row.SigningAlgorithm,
		NotValidAfter:        row.NotValidAfter,
		NotValidBefore:       row.NotValidBefore,
		Serial:               row.Serial,
		CertificateAuthority: row.CertificateAuthority,
		Source:               row.Source,
		Username:             row.Username,
		Path:                 row.Path,
	}
}
