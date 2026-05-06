-- name: UpsertHostOnOrbitEnroll :one
INSERT INTO hosts (
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
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
    @platform,
    @platform_like,
    @orbit_node_key::text,
    now(),
    now()
)
ON CONFLICT (hardware_uuid) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    hostname = EXCLUDED.hostname,
    computer_name = EXCLUDED.computer_name,
    hardware_serial = EXCLUDED.hardware_serial,
    hardware_model = EXCLUDED.hardware_model,
    platform = EXCLUDED.platform,
    platform_like = EXCLUDED.platform_like,
    orbit_node_key = EXCLUDED.orbit_node_key,
    enrolled_at = now(),
    last_seen_at = now(),
    updated_at = now(),
    deleted_at = NULL
RETURNING
    id,
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
    os_version,
    osquery_version,
    orbit_version,
    COALESCE(orbit_node_key, '')::text AS orbit_node_key,
    COALESCE(osquery_node_key, '')::text AS osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    enrolled_at,
    last_seen_at,
    detail_updated_at,
    created_at,
    updated_at;

-- name: UpsertHostOnOsqueryEnroll :one
INSERT INTO hosts (
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    os_version,
    platform,
    platform_like,
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
    @os_version,
    @platform,
    @platform_like,
    @osquery_version,
    @orbit_version,
    @osquery_node_key::text,
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
    os_version = EXCLUDED.os_version,
    platform = EXCLUDED.platform,
    platform_like = EXCLUDED.platform_like,
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
    last_seen_at = now(),
    updated_at = now(),
    deleted_at = NULL
RETURNING
    id,
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
    os_version,
    osquery_version,
    orbit_version,
    COALESCE(orbit_node_key, '')::text AS orbit_node_key,
    COALESCE(osquery_node_key, '')::text AS osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    enrolled_at,
    last_seen_at,
    detail_updated_at,
    created_at,
    updated_at;

-- name: ListHosts :many
SELECT
    id,
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
    os_version,
    osquery_version,
    orbit_version,
    COALESCE(orbit_node_key, '')::text AS orbit_node_key,
    COALESCE(osquery_node_key, '')::text AS osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    enrolled_at,
    last_seen_at,
    detail_updated_at,
    created_at,
    updated_at
FROM hosts
WHERE deleted_at IS NULL
ORDER BY last_seen_at DESC NULLS LAST, created_at DESC;

-- name: ListHostsBySoftwareTitle :many
SELECT
    id,
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
    os_version,
    osquery_version,
    orbit_version,
    COALESCE(orbit_node_key, '')::text AS orbit_node_key,
    COALESCE(osquery_node_key, '')::text AS osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    enrolled_at,
    last_seen_at,
    detail_updated_at,
    created_at,
    updated_at
FROM hosts
WHERE deleted_at IS NULL
    AND EXISTS (
        SELECT 1
        FROM host_software hs
        JOIN software s ON s.id = hs.software_id
        WHERE hs.host_id = hosts.id AND s.title_id = @software_title_id
    )
ORDER BY last_seen_at DESC NULLS LAST, created_at DESC;

-- name: ListHostsBySoftware :many
SELECT
    id,
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
    os_version,
    osquery_version,
    orbit_version,
    COALESCE(orbit_node_key, '')::text AS orbit_node_key,
    COALESCE(osquery_node_key, '')::text AS osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    enrolled_at,
    last_seen_at,
    detail_updated_at,
    created_at,
    updated_at
FROM hosts
WHERE deleted_at IS NULL
    AND EXISTS (
        SELECT 1
        FROM host_software hs
        WHERE hs.host_id = hosts.id AND hs.software_id = @software_id
    )
ORDER BY last_seen_at DESC NULLS LAST, created_at DESC;

-- name: GetHostByID :one
SELECT
    id,
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
    os_version,
    osquery_version,
    orbit_version,
    COALESCE(orbit_node_key, '')::text AS orbit_node_key,
    COALESCE(osquery_node_key, '')::text AS osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    enrolled_at,
    last_seen_at,
    detail_updated_at,
    created_at,
    updated_at
FROM hosts
WHERE id = @id AND deleted_at IS NULL;

-- name: TouchHostByOrbitNodeKey :one
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE orbit_node_key = @orbit_node_key::text AND deleted_at IS NULL
RETURNING
    id,
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
    os_version,
    osquery_version,
    orbit_version,
    COALESCE(orbit_node_key, '')::text AS orbit_node_key,
    COALESCE(osquery_node_key, '')::text AS osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    enrolled_at,
    last_seen_at,
    detail_updated_at,
    created_at,
    updated_at;

-- name: TouchHostByOsqueryNodeKey :one
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE osquery_node_key = @osquery_node_key::text AND deleted_at IS NULL
RETURNING
    id,
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model,
    platform,
    platform_like,
    os_version,
    osquery_version,
    orbit_version,
    COALESCE(orbit_node_key, '')::text AS orbit_node_key,
    COALESCE(osquery_node_key, '')::text AS osquery_node_key,
    cpu_brand,
    cpu_logical_cores,
    cpu_physical_cores,
    physical_memory,
    hardware_vendor,
    kernel_version,
    enrolled_at,
    last_seen_at,
    detail_updated_at,
    created_at,
    updated_at;

-- name: ApplyHostDetail :exec
UPDATE hosts
SET
    hostname = COALESCE(NULLIF(@hostname::text, ''), hostname),
    computer_name = COALESCE(NULLIF(@computer_name::text, ''), computer_name),
    display_name = COALESCE(NULLIF(@computer_name::text, ''), NULLIF(@hostname::text, ''), display_name),
    hardware_serial = COALESCE(NULLIF(@hardware_serial::text, ''), hardware_serial),
    hardware_model = COALESCE(NULLIF(@hardware_model::text, ''), hardware_model),
    os_version = COALESCE(NULLIF(@os_version::text, ''), os_version),
    platform = COALESCE(NULLIF(@platform::text, ''), platform),
    platform_like = COALESCE(NULLIF(@platform_like::text, ''), platform_like),
    osquery_version = COALESCE(NULLIF(@osquery_version::text, ''), osquery_version),
    orbit_version = COALESCE(NULLIF(@orbit_version::text, ''), orbit_version),
    cpu_brand = COALESCE(NULLIF(@cpu_brand::text, ''), cpu_brand),
    cpu_logical_cores = CASE WHEN @cpu_logical_cores::integer > 0 THEN @cpu_logical_cores::integer ELSE cpu_logical_cores END,
    cpu_physical_cores = CASE WHEN @cpu_physical_cores::integer > 0 THEN @cpu_physical_cores::integer ELSE cpu_physical_cores END,
    physical_memory = CASE WHEN @physical_memory::bigint > 0 THEN @physical_memory::bigint ELSE physical_memory END,
    hardware_vendor = COALESCE(NULLIF(@hardware_vendor::text, ''), hardware_vendor),
    kernel_version = COALESCE(NULLIF(@kernel_version::text, ''), kernel_version),
    updated_at = now()
WHERE id = @id AND deleted_at IS NULL;

-- name: MarkHostDetailFresh :exec
UPDATE hosts
SET detail_updated_at = now(), updated_at = now()
WHERE id = @id AND deleted_at IS NULL;
