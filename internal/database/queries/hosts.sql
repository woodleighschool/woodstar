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
    platform = EXCLUDED.platform,
    platform_like = EXCLUDED.platform_like,
    orbit_node_key = EXCLUDED.orbit_node_key,
    enrolled_at = now(),
    last_seen_at = now(),
    updated_at = now(),
    deleted_at = NULL
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
    @hardware_version,
    @os_version,
    @os_name,
    @os_build,
    @platform,
    @platform_like,
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
    detail_query_hash = '',
    last_seen_at = now(),
    updated_at = now(),
    deleted_at = NULL
RETURNING *;

-- name: ListHosts :many
SELECT *
FROM hosts
WHERE deleted_at IS NULL
    AND (
        @q::text = ''
        OR display_name ILIKE '%' || @q::text || '%'
        OR hostname ILIKE '%' || @q::text || '%'
        OR computer_name ILIKE '%' || @q::text || '%'
        OR hardware_serial ILIKE '%' || @q::text || '%'
        OR hardware_uuid ILIKE '%' || @q::text || '%'
        OR hardware_model ILIKE '%' || @q::text || '%'
        OR os_version ILIKE '%' || @q::text || '%'
        OR EXISTS (
            SELECT 1 FROM host_emails he
            WHERE he.host_id = hosts.id AND he.email ILIKE '%' || @q::text || '%'
        )
    )
    AND (
        @platform::text = ''
        OR platform = @platform::text
        OR (@platform::text = 'darwin' AND platform IN ('darwin', 'macos'))
        OR (@platform::text = 'linux' AND platform <> '' AND platform NOT IN ('darwin', 'macos', 'windows', 'chrome'))
    )
    AND (
        @status::text = ''
        OR (@status::text = 'online' AND last_seen_at >= now() - interval '5 minutes')
        OR (@status::text = 'offline' AND (last_seen_at IS NULL OR last_seen_at < now() - interval '5 minutes'))
    )
    AND (@label_id::bigint = 0 OR EXISTS (
        SELECT 1 FROM label_membership lm
        WHERE lm.host_id = hosts.id AND lm.label_id = @label_id::bigint
    ))
    AND (@software_id::bigint = 0 OR EXISTS (
        SELECT 1 FROM host_software hs
        WHERE hs.host_id = hosts.id AND hs.software_id = @software_id::bigint
    ))
    AND (@software_title_id::bigint = 0 OR EXISTS (
        SELECT 1
        FROM host_software hs
        JOIN software s ON s.id = hs.software_id
        WHERE hs.host_id = hosts.id AND s.title_id = @software_title_id::bigint
    ))
ORDER BY
    CASE WHEN @order_key::text = 'platform' AND @order_direction::text = 'asc' THEN lower(platform) END ASC,
    CASE WHEN @order_key::text = 'platform' AND @order_direction::text = 'desc' THEN lower(platform) END DESC,
    CASE WHEN @order_key::text = 'hardware_serial' AND @order_direction::text = 'asc' THEN lower(hardware_serial) END ASC,
    CASE WHEN @order_key::text = 'hardware_serial' AND @order_direction::text = 'desc' THEN lower(hardware_serial) END DESC,
    CASE WHEN @order_key::text = 'os_version' AND @order_direction::text = 'asc' THEN lower(os_version) END ASC,
    CASE WHEN @order_key::text = 'os_version' AND @order_direction::text = 'desc' THEN lower(os_version) END DESC,
    CASE WHEN @order_key::text = 'last_seen_at' AND @order_direction::text = 'asc' THEN last_seen_at END ASC NULLS LAST,
    CASE WHEN @order_key::text = 'last_seen_at' AND @order_direction::text = 'desc' THEN last_seen_at END DESC NULLS LAST,
    CASE WHEN @order_key::text = 'display_name' AND @order_direction::text = 'desc' THEN lower(display_name) END DESC,
    lower(display_name),
    id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountHosts :one
SELECT count(*)::integer
FROM hosts
WHERE deleted_at IS NULL
    AND (
        @q::text = ''
        OR display_name ILIKE '%' || @q::text || '%'
        OR hostname ILIKE '%' || @q::text || '%'
        OR computer_name ILIKE '%' || @q::text || '%'
        OR hardware_serial ILIKE '%' || @q::text || '%'
        OR hardware_uuid ILIKE '%' || @q::text || '%'
        OR hardware_model ILIKE '%' || @q::text || '%'
        OR os_version ILIKE '%' || @q::text || '%'
        OR EXISTS (
            SELECT 1 FROM host_emails he
            WHERE he.host_id = hosts.id AND he.email ILIKE '%' || @q::text || '%'
        )
    )
    AND (
        @platform::text = ''
        OR platform = @platform::text
        OR (@platform::text = 'darwin' AND platform IN ('darwin', 'macos'))
        OR (@platform::text = 'linux' AND platform <> '' AND platform NOT IN ('darwin', 'macos', 'windows', 'chrome'))
    )
    AND (
        @status::text = ''
        OR (@status::text = 'online' AND last_seen_at >= now() - interval '5 minutes')
        OR (@status::text = 'offline' AND (last_seen_at IS NULL OR last_seen_at < now() - interval '5 minutes'))
    )
    AND (@label_id::bigint = 0 OR EXISTS (
        SELECT 1 FROM label_membership lm
        WHERE lm.host_id = hosts.id AND lm.label_id = @label_id::bigint
    ))
    AND (@software_id::bigint = 0 OR EXISTS (
        SELECT 1 FROM host_software hs
        WHERE hs.host_id = hosts.id AND hs.software_id = @software_id::bigint
    ))
    AND (@software_title_id::bigint = 0 OR EXISTS (
        SELECT 1
        FROM host_software hs
        JOIN software s ON s.id = hs.software_id
        WHERE hs.host_id = hosts.id AND s.title_id = @software_title_id::bigint
    ));

-- name: GetHostByID :one
SELECT *
FROM hosts
WHERE id = @id AND deleted_at IS NULL;

-- name: TouchHostByOrbitNodeKey :one
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE orbit_node_key = @orbit_node_key AND orbit_node_key <> '' AND deleted_at IS NULL
RETURNING *;

-- name: TouchHostByOsqueryNodeKey :one
UPDATE hosts
SET last_seen_at = now(), updated_at = now()
WHERE osquery_node_key = @osquery_node_key AND osquery_node_key <> '' AND deleted_at IS NULL
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
    platform = COALESCE(NULLIF(@platform::text, ''), platform),
    platform_like = COALESCE(NULLIF(@platform_like::text, ''), platform_like),
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
    uptime_seconds = COALESCE(sqlc.narg(uptime_seconds)::bigint, uptime_seconds),
    last_restarted_at = COALESCE(sqlc.narg(last_restarted_at)::timestamptz, last_restarted_at),
    disk_space_available_bytes = COALESCE(sqlc.narg(disk_space_available_bytes)::bigint, disk_space_available_bytes),
    disk_space_total_bytes = COALESCE(sqlc.narg(disk_space_total_bytes)::bigint, disk_space_total_bytes),
    public_ip = COALESCE(NULLIF(@public_ip::text, '')::inet, public_ip),
    primary_ip = COALESCE(NULLIF(@primary_ip::text, '')::inet, primary_ip),
    primary_mac = COALESCE(NULLIF(@primary_mac::text, ''), primary_mac),
    distributed_interval = COALESCE(sqlc.narg(distributed_interval)::integer, distributed_interval),
    config_tls_refresh = COALESCE(sqlc.narg(config_tls_refresh)::integer, config_tls_refresh),
    updated_at = now()
WHERE id = @id AND deleted_at IS NULL;

-- name: MarkHostDetailFresh :exec
UPDATE hosts
SET detail_updated_at = now(), detail_query_hash = @detail_query_hash, updated_at = now()
WHERE id = @id AND deleted_at IS NULL;

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

-- name: AddHostToAllHostsLabel :exec
INSERT INTO label_membership (label_id, host_id)
SELECT id, @host_id
FROM labels
WHERE name = 'All Hosts' AND label_type = 'builtin' AND label_membership_type = 'manual'
ON CONFLICT (label_id, host_id) DO NOTHING;
