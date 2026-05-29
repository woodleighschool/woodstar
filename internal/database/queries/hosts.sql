-- name: UpsertHostOnOrbitEnroll :one
INSERT INTO hosts (
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    orbit_node_key,
    enrolled_at,
    last_seen_at
)
VALUES (
    @hardware_uuid,
    @display_name,
    @hostname,
    @computer_name,
    @hardware_serial,
    @hardware_model,
    @orbit_node_key,
    now(),
    now()
)
ON CONFLICT (hardware_uuid) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    hostname = EXCLUDED.hostname,
    computer_name = EXCLUDED.computer_name,
    hardware_serial = EXCLUDED.hardware_serial,
    hardware_model = EXCLUDED.hardware_model,
    orbit_node_key = EXCLUDED.orbit_node_key,
    enrolled_at = now(),
    last_seen_at = now(),
    updated_at = now()
RETURNING *;

-- name: UpsertHostOnOsqueryEnroll :one
INSERT INTO hosts (
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    hardware_version,
    os_version,
    os_name,
    os_build,
    osquery_version,
    orbit_version,
    osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    last_seen_at,
    detail_updated_at
)
VALUES (
    @hardware_uuid,
    @display_name,
    @hostname,
    @computer_name,
    @hardware_serial,
    @hardware_model,
    @hardware_version,
    @os_version,
    @os_name,
    @os_build,
    @osquery_version,
    @orbit_version,
    @osquery_node_key,
    @cpu_brand,
    @cpu_logical_cores,
    @cpu_physical_cores,
    @physical_memory,
    @hardware_vendor,
    @kernel_version,
    now(),
    NULL
)
ON CONFLICT (hardware_uuid) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    hostname = EXCLUDED.hostname,
    computer_name = EXCLUDED.computer_name,
    hardware_serial = EXCLUDED.hardware_serial,
    hardware_model = EXCLUDED.hardware_model,
    hardware_version = COALESCE(NULLIF(EXCLUDED.hardware_version, ''), hosts.hardware_version),
    os_version = EXCLUDED.os_version,
    os_name = COALESCE(NULLIF(EXCLUDED.os_name, ''), hosts.os_name),
    os_build = COALESCE(NULLIF(EXCLUDED.os_build, ''), hosts.os_build),
    osquery_version = EXCLUDED.osquery_version,
    orbit_version = COALESCE(NULLIF(EXCLUDED.orbit_version, ''), hosts.orbit_version),
    osquery_node_key = EXCLUDED.osquery_node_key,
    cpu_brand = EXCLUDED.cpu_brand,
    cpu_logical_cores = EXCLUDED.cpu_logical_cores,
    cpu_physical_cores = EXCLUDED.cpu_physical_cores,
    physical_memory = EXCLUDED.physical_memory,
    hardware_vendor = EXCLUDED.hardware_vendor,
    kernel_version = EXCLUDED.kernel_version,
    detail_updated_at = NULL,
    detail_query_hash = '',
    last_seen_at = now(),
    updated_at = now()
RETURNING *;

-- name: GetHostByID :one
SELECT *
FROM hosts
WHERE id = @id;

-- name: DeleteHost :one
DELETE FROM hosts
WHERE id = @id
RETURNING id;

-- name: DeleteHosts :many
DELETE FROM hosts
WHERE id = ANY(@ids::bigint[])
RETURNING id;

-- name: TouchHostByOrbitNodeKey :one
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE orbit_node_key = @orbit_node_key AND orbit_node_key <> ''
RETURNING *;

-- name: TouchHostByOsqueryNodeKey :one
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE osquery_node_key = @osquery_node_key AND osquery_node_key <> ''
RETURNING *;

-- name: ApplyHostDetail :exec
UPDATE hosts
SET
    hostname = COALESCE(NULLIF(@hostname::text, ''), hostname),
    computer_name = COALESCE(NULLIF(@computer_name::text, ''), computer_name),
    display_name = COALESCE(NULLIF(@computer_name::text, ''), NULLIF(@hostname::text, ''), display_name),
    hardware_serial = COALESCE(NULLIF(@hardware_serial::text, ''), hardware_serial),
    hardware_model = COALESCE(NULLIF(@hardware_model::text, ''), hardware_model),
    hardware_version = COALESCE(NULLIF(@hardware_version::text, ''), hardware_version),
    os_name = COALESCE(NULLIF(@os_name::text, ''), os_name),
    os_version = COALESCE(NULLIF(@os_version::text, ''), os_version),
    os_build = COALESCE(NULLIF(@os_build::text, ''), os_build),
    osquery_version = COALESCE(NULLIF(@osquery_version::text, ''), osquery_version),
    orbit_version = COALESCE(NULLIF(@orbit_version::text, ''), orbit_version),
    cpu_type = COALESCE(NULLIF(@cpu_type::text, ''), cpu_type),
    cpu_subtype = COALESCE(NULLIF(@cpu_subtype::text, ''), cpu_subtype),
    cpu_brand = COALESCE(NULLIF(@cpu_brand::text, ''), cpu_brand),
    cpu_logical_cores = CASE WHEN @cpu_logical_cores::integer > 0 THEN @cpu_logical_cores::integer ELSE cpu_logical_cores END,
    cpu_physical_cores = CASE WHEN @cpu_physical_cores::integer > 0 THEN @cpu_physical_cores::integer ELSE cpu_physical_cores END,
    physical_memory = CASE WHEN @physical_memory::bigint > 0 THEN @physical_memory::bigint ELSE physical_memory END,
    hardware_vendor = COALESCE(NULLIF(@hardware_vendor::text, ''), hardware_vendor),
    kernel_version = COALESCE(NULLIF(@kernel_version::text, ''), kernel_version),
    last_restarted_at = COALESCE(sqlc.narg(last_restarted_at)::timestamptz, last_restarted_at),
    disk_space_available_bytes = COALESCE(sqlc.narg(disk_space_available_bytes)::bigint, disk_space_available_bytes),
    disk_space_total_bytes = COALESCE(sqlc.narg(disk_space_total_bytes)::bigint, disk_space_total_bytes),
    public_ip = COALESCE(NULLIF(@public_ip::text, '')::inet, public_ip),
    primary_ip = COALESCE(NULLIF(@primary_ip::text, '')::inet, primary_ip),
    primary_mac = COALESCE(NULLIF(@primary_mac::text, ''), primary_mac),
    distributed_interval = COALESCE(sqlc.narg(distributed_interval)::integer, distributed_interval),
    config_tls_refresh = COALESCE(sqlc.narg(config_tls_refresh)::integer, config_tls_refresh),
    updated_at = now()
WHERE id = @id;

-- name: MarkHostDetailFresh :exec
UPDATE hosts
SET detail_updated_at = now(), detail_query_hash = @detail_query_hash, updated_at = now()
WHERE id = @id;

-- name: DeleteHostUsers :exec
DELETE FROM host_users
WHERE host_id = @host_id;

-- name: InsertHostUser :exec
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
    updated_at = now();

-- name: ListHostUsers :many
SELECT *
FROM host_users
WHERE host_id = @host_id
ORDER BY username, uid, id;

-- name: DeleteHostBatteries :exec
DELETE FROM host_batteries
WHERE host_id = @host_id;

-- name: InsertHostBattery :exec
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
    updated_at = now();

-- name: ListHostBatteries :many
SELECT *
FROM host_batteries
WHERE host_id = @host_id
ORDER BY serial_number, id;

-- name: DeleteHostCertificates :exec
DELETE FROM host_certificates
WHERE host_id = @host_id;

-- name: InsertHostCertificate :exec
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
    updated_at = now();

-- name: ListHostCertificates :many
SELECT *
FROM host_certificates
WHERE host_id = @host_id
ORDER BY common_name, sha1, id;

-- name: AddHostToAllHostsLabel :exec
INSERT INTO label_membership (label_id, host_id)
SELECT id, @host_id
FROM labels
WHERE name = 'All Hosts' AND label_type = 'builtin' AND label_membership_type = 'manual'
ON CONFLICT (label_id, host_id) DO NOTHING;

-- name: ListSelectedHostIDs :many
SELECT id
FROM hosts
WHERE id = ANY(@host_ids::bigint[])
ORDER BY id;

-- name: ListOnlineSelectedHostIDs :many
SELECT id
FROM hosts
WHERE id = ANY(@host_ids::bigint[])
  AND last_seen_at >= @online_since
ORDER BY id;

-- name: CountSelectedHostStatus :one
SELECT
    count(*)::integer AS total,
    count(*) FILTER (WHERE last_seen_at >= @online_since)::integer AS online,
    count(*) FILTER (WHERE last_seen_at IS NULL OR last_seen_at < @online_since)::integer AS offline
FROM hosts
WHERE id = ANY(@host_ids::bigint[]);

-- name: ListSelectedLabels :many
SELECT id, name, label_type
FROM labels
WHERE id = ANY(@label_ids::bigint[])
ORDER BY id;

-- name: ListAllHostIDs :many
SELECT id
FROM hosts
ORDER BY id;

-- name: ListHostIDsByAnyLabel :many
SELECT DISTINCT h.id
FROM hosts h
JOIN label_membership lm ON lm.host_id = h.id
WHERE lm.label_id = ANY(@label_ids::bigint[])
ORDER BY h.id;

-- name: ListHostIDsByBuiltinAndRegularLabels :many
SELECT DISTINCT h.id
FROM hosts h
WHERE EXISTS (
        SELECT 1
        FROM label_membership lm
        WHERE lm.host_id = h.id AND lm.label_id = ANY(@builtin_label_ids::bigint[])
    )
  AND EXISTS (
        SELECT 1
        FROM label_membership lm
        WHERE lm.host_id = h.id AND lm.label_id = ANY(@regular_label_ids::bigint[])
    )
ORDER BY h.id;
