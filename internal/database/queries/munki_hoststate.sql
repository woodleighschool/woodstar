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
    @success,
    @errors,
    @warnings,
    @problem_installs,
    sqlc.narg(run_started_at)::timestamptz,
    sqlc.narg(run_ended_at)::timestamptz,
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
    sqlc.narg(run_ended_at)::timestamptz,
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
