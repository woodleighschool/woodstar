-- name: CreateMunkiSoftwareTitle :one
INSERT INTO munki_software_titles (
    name,
    display_name,
    description,
    category,
    developer
)
VALUES (
    @name,
    @display_name,
    @description,
    @category,
    @developer
)
RETURNING *;

-- name: ListMunkiSoftwareTitles :many
SELECT *
FROM munki_software_titles
ORDER BY lower(COALESCE(NULLIF(display_name, ''), name)), lower(name), id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountMunkiSoftwareTitles :one
SELECT COUNT(*)::integer
FROM munki_software_titles;

-- name: GetMunkiSoftwareTitleByID :one
SELECT *
FROM munki_software_titles
WHERE id = @id;

-- name: CreateMunkiArtifact :one
INSERT INTO munki_artifacts (
    kind,
    display_name,
    location,
    content_type,
    size_bytes,
    sha256,
    storage_key
)
VALUES (
    @kind::munki_artifact_kind,
    @display_name,
    @location,
    @content_type,
    @size_bytes,
    @sha256,
    @storage_key
)
RETURNING *;

-- name: ListMunkiArtifacts :many
SELECT *
FROM munki_artifacts
ORDER BY lower(COALESCE(NULLIF(display_name, ''), location)), lower(location), id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountMunkiArtifacts :one
SELECT COUNT(*)::integer
FROM munki_artifacts;

-- name: GetMunkiArtifactByID :one
SELECT *
FROM munki_artifacts
WHERE id = @id;

-- name: GetMunkiArtifactByKindAndLocation :one
SELECT *
FROM munki_artifacts
WHERE kind = @kind::munki_artifact_kind
  AND location = @location;

-- name: CreateMunkiRelease :one
INSERT INTO munki_releases (
    software_id,
    name,
    version,
    display_name,
    pkginfo,
    installer_artifact_id,
    eligible
)
VALUES (
    @software_id,
    @name,
    @version,
    @display_name,
    @pkginfo::jsonb,
    sqlc.narg(installer_artifact_id)::bigint,
    @eligible
)
RETURNING *;

-- name: ListMunkiReleases :many
SELECT *
FROM munki_releases
ORDER BY lower(COALESCE(NULLIF(display_name, ''), name)), lower(name), lower(version), id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountMunkiReleases :one
SELECT COUNT(*)::integer
FROM munki_releases;

-- name: GetMunkiReleaseByID :one
SELECT *
FROM munki_releases
WHERE id = @id;

-- name: CreateMunkiAssignment :one
INSERT INTO munki_assignments (
    release_id,
    intent,
    all_hosts
)
VALUES (
    @release_id,
    @intent::munki_assignment_intent,
    @all_hosts
)
RETURNING *;

-- name: InsertMunkiAssignmentIncludeLabels :exec
INSERT INTO munki_assignment_include_labels (
    assignment_id,
    label_id
)
SELECT @assignment_id, unnest(@label_ids::bigint[]);

-- name: InsertMunkiAssignmentExcludeLabels :exec
INSERT INTO munki_assignment_exclude_labels (
    assignment_id,
    label_id
)
SELECT @assignment_id, unnest(@label_ids::bigint[]);

-- name: InsertMunkiAssignmentIncludeHosts :exec
INSERT INTO munki_assignment_include_hosts (
    assignment_id,
    host_id
)
SELECT @assignment_id, unnest(@host_ids::bigint[]);

-- name: InsertMunkiAssignmentExcludeHosts :exec
INSERT INTO munki_assignment_exclude_hosts (
    assignment_id,
    host_id
)
SELECT @assignment_id, unnest(@host_ids::bigint[]);

-- name: ListMunkiAssignments :many
SELECT *
FROM munki_assignments
ORDER BY id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountMunkiAssignments :one
SELECT COUNT(*)::integer
FROM munki_assignments;

-- name: ListMunkiAssignmentIncludeLabelIDs :many
SELECT label_id
FROM munki_assignment_include_labels
WHERE assignment_id = @assignment_id
ORDER BY label_id;

-- name: ListMunkiAssignmentExcludeLabelIDs :many
SELECT label_id
FROM munki_assignment_exclude_labels
WHERE assignment_id = @assignment_id
ORDER BY label_id;

-- name: ListMunkiAssignmentIncludeHostIDs :many
SELECT host_id
FROM munki_assignment_include_hosts
WHERE assignment_id = @assignment_id
ORDER BY host_id;

-- name: ListMunkiAssignmentExcludeHostIDs :many
SELECT host_id
FROM munki_assignment_exclude_hosts
WHERE assignment_id = @assignment_id
ORDER BY host_id;

-- name: ListEffectiveMunkiReleasesForHost :many
SELECT
    a.id AS assignment_id,
    a.intent,
    r.id AS release_id,
    r.software_id,
    r.name,
    r.version,
    r.display_name,
    r.pkginfo,
    r.installer_artifact_id,
    art.location AS installer_artifact_location,
    CASE
        WHEN EXISTS (
            SELECT 1
            FROM munki_assignment_include_hosts ih
            WHERE ih.assignment_id = a.id AND ih.host_id = @host_id
        ) THEN 30
        WHEN EXISTS (
            SELECT 1
            FROM munki_assignment_include_labels il
            JOIN label_membership lm ON lm.label_id = il.label_id
            WHERE il.assignment_id = a.id AND lm.host_id = @host_id
        ) THEN 20
        WHEN a.all_hosts THEN 10
        ELSE 0
    END AS scope_rank
FROM munki_assignments a
JOIN munki_releases r ON r.id = a.release_id
LEFT JOIN munki_artifacts art ON art.id = r.installer_artifact_id
WHERE r.eligible
  AND (
    a.all_hosts
    OR EXISTS (
      SELECT 1
      FROM munki_assignment_include_hosts ih
      WHERE ih.assignment_id = a.id AND ih.host_id = @host_id
    )
    OR EXISTS (
      SELECT 1
      FROM munki_assignment_include_labels il
      JOIN label_membership lm ON lm.label_id = il.label_id
      WHERE il.assignment_id = a.id AND lm.host_id = @host_id
    )
  )
  AND NOT EXISTS (
    SELECT 1
    FROM munki_assignment_exclude_hosts eh
    WHERE eh.assignment_id = a.id AND eh.host_id = @host_id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM munki_assignment_exclude_labels el
    JOIN label_membership lm ON lm.label_id = el.label_id
    WHERE el.assignment_id = a.id AND lm.host_id = @host_id
  )
ORDER BY lower(r.name), r.id, a.id;

-- name: UpsertMunkiHostStatus :exec
INSERT INTO munki_host_status (
    host_id,
    version,
    manifest_name,
    success,
    errors,
    warnings,
    problem_installs,
    run_started_at,
    run_ended_at,
    last_seen_at
)
VALUES (
    @host_id,
    @version,
    @manifest_name,
    sqlc.narg(success)::boolean,
    @errors,
    @warnings,
    @problem_installs,
    @run_started_at,
    @run_ended_at,
    now()
)
ON CONFLICT (host_id) DO UPDATE SET
    version = EXCLUDED.version,
    manifest_name = EXCLUDED.manifest_name,
    success = EXCLUDED.success,
    errors = EXCLUDED.errors,
    warnings = EXCLUDED.warnings,
    problem_installs = EXCLUDED.problem_installs,
    run_started_at = EXCLUDED.run_started_at,
    run_ended_at = EXCLUDED.run_ended_at,
    last_seen_at = now(),
    updated_at = now();

-- name: ClearMunkiHostStatus :exec
DELETE FROM munki_host_status
WHERE host_id = @host_id;

-- name: DeleteMunkiHostItems :exec
DELETE FROM munki_host_items
WHERE host_id = @host_id;

-- name: InsertMunkiHostItem :exec
INSERT INTO munki_host_items (
    host_id,
    name,
    installed,
    installed_version,
    run_ended_at,
    last_seen_at
)
VALUES (
    @host_id,
    @name,
    @installed,
    @installed_version,
    @run_ended_at,
    now()
)
ON CONFLICT (host_id, name) DO UPDATE SET
    installed = EXCLUDED.installed,
    installed_version = EXCLUDED.installed_version,
    run_ended_at = EXCLUDED.run_ended_at,
    last_seen_at = now(),
    updated_at = now();

-- name: ListMunkiHostItems :many
SELECT *
FROM munki_host_items
WHERE host_id = @host_id
ORDER BY lower(name), name;

-- name: GetMunkiHostStatus :one
SELECT *
FROM munki_host_status
WHERE host_id = @host_id;
