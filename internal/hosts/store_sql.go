package hosts

import "time"

// hostColumnsSQL is the canonical hosts projection shared by Get and List.
const hostColumnsSQL = `
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

const hostSelectSQL = `SELECT` + hostColumnsSQL + `
FROM hosts`

type orbitEnrollWrite struct {
	HardwareUUID            string `db:"hardware_uuid"`
	DisplayName             string `db:"display_name"`
	Hostname                string `db:"hostname"`
	ComputerName            string `db:"computer_name"`
	HardwareSerial          string `db:"hardware_serial"`
	HardwareModelIdentifier string `db:"hardware_model_identifier"`
	OrbitNodeKey            string `db:"orbit_node_key"`
}

const upsertHostOnOrbitEnrollSQL = `
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
	enrollment_agent = EXCLUDED.enrollment_agent,
	enrolled_at = now(),
	last_seen_at = now(),
	updated_at = now()
RETURNING` + hostColumnsSQL

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

const upsertHostOnOsqueryEnrollSQL = `
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
RETURNING` + hostColumnsSQL

const addHostToAllHostsLabelSQL = `
INSERT INTO label_membership (label_id, host_id)
SELECT id, @host_id
FROM labels
WHERE builtin_key = @builtin_key::text AND label_type = 'builtin' AND label_membership_type = 'manual'
ON CONFLICT (label_id, host_id) DO NOTHING`

const touchHostByOrbitNodeKeySQL = `
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE orbit_node_key = $1 AND orbit_node_key <> ''
RETURNING` + hostColumnsSQL

const touchHostByOsqueryNodeKeySQL = `
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE osquery_node_key = $1 AND osquery_node_key <> ''
RETURNING` + hostColumnsSQL

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

const applyHostInventorySQL = `
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
WHERE id = @id`

const markHostInventoryFreshSQL = `
UPDATE hosts
SET inventory_updated_at = now(), inventory_query_hash = @inventory_query_hash, updated_at = now()
WHERE id = @id`

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

const deleteHostUsersSQL = `DELETE FROM host_users WHERE host_id = $1`

const insertHostUserSQL = `
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
	shell = EXCLUDED.shell,
	updated_at = now()`

const listHostUsersSQL = `
SELECT id, host_id, uid, username, type, description, directory, shell, created_at, updated_at
FROM host_users
WHERE host_id = $1
ORDER BY username, uid, id`

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

const deleteHostBatteriesSQL = `DELETE FROM host_batteries WHERE host_id = $1`

const insertHostBatterySQL = `
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
	percent_remaining = EXCLUDED.percent_remaining,
	updated_at = now()`

const listHostBatteriesSQL = `
SELECT
	id, host_id, serial_number, manufacturer, model, chemistry, cycle_count,
	health, designed_capacity, max_capacity, current_capacity, percent_remaining,
	created_at, updated_at
FROM host_batteries
WHERE host_id = $1
ORDER BY serial_number, id`

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

const deleteHostCertificatesSQL = `DELETE FROM host_certificates WHERE host_id = $1`

const insertHostCertificateSQL = `
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
	path = EXCLUDED.path,
	updated_at = now()`

const listHostCertificatesSQL = `
SELECT
	id, host_id, sha1, common_name,
	subject_country, subject_organization, subject_organizational_unit, subject_common_name,
	issuer_country, issuer_organization, issuer_organizational_unit, issuer_common_name,
	key_algorithm, key_strength, key_usage, signing_algorithm,
	not_valid_after, not_valid_before, serial, certificate_authority,
	source, username, path, created_at, updated_at
FROM host_certificates
WHERE host_id = $1
ORDER BY common_name, sha1, id`
