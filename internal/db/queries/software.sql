-- name: DeleteHostSoftwarePaths :exec
DELETE FROM host_software_installed_paths
WHERE host_id = @host_id;

-- name: DeleteHostSoftware :exec
DELETE FROM host_software
WHERE host_id = @host_id;

-- name: UpsertSoftwareTitleByBundle :one
INSERT INTO software_titles (
    name,
    display_name,
    source,
    extension_for,
    bundle_identifier,
    vendor
)
VALUES (
    @name,
    @display_name,
    @source,
    @extension_for,
    @bundle_identifier,
    @vendor
)
ON CONFLICT (bundle_identifier, source, extension_for)
WHERE bundle_identifier <> ''
DO UPDATE SET
    vendor = COALESCE(NULLIF(EXCLUDED.vendor, ''), software_titles.vendor),
    updated_at = now()
RETURNING *;

-- name: UpsertSoftwareTitleByName :one
INSERT INTO software_titles (
    name,
    display_name,
    source,
    extension_for,
    bundle_identifier,
    vendor
)
VALUES (
    @name,
    @display_name,
    @source,
    @extension_for,
    @bundle_identifier,
    @vendor
)
ON CONFLICT (name, source, extension_for, bundle_identifier) DO UPDATE SET
    vendor = COALESCE(NULLIF(EXCLUDED.vendor, ''), software_titles.vendor),
    updated_at = now()
RETURNING *;

-- name: UpsertSoftware :one
INSERT INTO software (
    title_id,
    name,
    version,
    source,
    bundle_identifier,
    extension_id,
    extension_for,
    vendor,
    arch,
    release
)
VALUES (
    @title_id,
    @name,
    @version,
    @source,
    @bundle_identifier,
    @extension_id,
    @extension_for,
    @vendor,
    @arch,
    @release
)
ON CONFLICT (
    title_id,
    version,
    source,
    bundle_identifier,
    extension_id,
    extension_for,
    vendor,
    arch,
    release
) DO UPDATE SET
    updated_at = now()
RETURNING *;

-- name: UpsertHostSoftware :exec
INSERT INTO host_software (
    host_id,
    software_id,
    last_seen_at,
    last_opened_at
)
VALUES (
    @host_id,
    @software_id,
    now(),
    @last_opened_at
)
ON CONFLICT (host_id, software_id) DO UPDATE SET
    last_seen_at = now(),
    last_opened_at = EXCLUDED.last_opened_at;

-- name: InsertHostSoftwareInstalledPath :exec
INSERT INTO host_software_installed_paths (
    host_id,
    software_id,
    installed_path,
    team_identifier,
    cdhash_sha256,
    executable_sha256,
    executable_path,
    last_seen_at
)
VALUES (
    @host_id,
    @software_id,
    @installed_path,
    @team_identifier,
    NULLIF(@cdhash_sha256::text, ''),
    NULLIF(@executable_sha256::text, ''),
    NULLIF(@executable_path::text, ''),
    now()
)
ON CONFLICT (host_id, software_id, installed_path) DO UPDATE SET
    team_identifier = EXCLUDED.team_identifier,
    cdhash_sha256 = EXCLUDED.cdhash_sha256,
    executable_sha256 = EXCLUDED.executable_sha256,
    executable_path = EXCLUDED.executable_path,
    last_seen_at = now();
