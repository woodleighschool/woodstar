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

-- name: GetSoftwareTitleSummary :one
SELECT
    st.id,
    st.name,
    st.display_name,
    st.source,
    st.extension_for,
    st.bundle_identifier,
    st.vendor,
    COUNT(DISTINCT hs.host_id)::integer AS hosts_count,
    COUNT(DISTINCT s.id)::integer AS versions_count,
    MAX(hs.last_seen_at) AS counts_updated_at
FROM software_titles st
LEFT JOIN software s ON s.title_id = st.id
LEFT JOIN host_software hs ON hs.software_id = s.id
WHERE st.id = @id
GROUP BY st.id;

-- name: ListSoftwareTitleVersions :many
SELECT
    s.title_id,
    s.id,
    s.version,
    s.bundle_identifier,
    COUNT(DISTINCT hs.host_id)::integer AS hosts_count
FROM software s
LEFT JOIN host_software hs ON hs.software_id = s.id
WHERE s.title_id = ANY(@title_ids::bigint[])
GROUP BY s.id
ORDER BY array_position(@title_ids::bigint[], s.title_id), lower(s.version), s.id;

-- name: ListHostSoftwareRows :many
SELECT
    st.id AS title_id,
    st.name AS title_name,
    st.display_name,
    st.source,
    st.extension_for,
    s.id AS software_id,
    s.version,
    s.bundle_identifier,
    hs.last_opened_at,
    COALESCE(paths.installed_path, '') AS installed_path,
    COALESCE(paths.team_identifier, '') AS team_identifier,
    COALESCE(paths.cdhash_sha256, '') AS cdhash_sha256,
    COALESCE(paths.executable_sha256, '') AS executable_sha256,
    COALESCE(paths.executable_path, '') AS executable_path
FROM host_software hs
JOIN software s ON s.id = hs.software_id
JOIN software_titles st ON st.id = s.title_id
LEFT JOIN host_software_installed_paths paths
    ON paths.host_id = hs.host_id AND paths.software_id = hs.software_id
WHERE hs.host_id = @host_id
  AND st.id = ANY(@title_ids::bigint[])
ORDER BY array_position(@title_ids::bigint[], st.id), lower(s.version), paths.installed_path;
