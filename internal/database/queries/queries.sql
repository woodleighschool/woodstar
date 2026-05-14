-- name: GetSavedQueryByID :one
SELECT
    id,
    name,
    description,
    query,
    platform,
    min_osquery_version,
    schedule_interval,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at
FROM queries
WHERE id = @id;

-- name: CreateSavedQuery :one
INSERT INTO queries (
    name,
    description,
    query,
    platform,
    min_osquery_version,
    schedule_interval,
    created_by_user_id
)
VALUES (
    @name,
    @description,
    @query,
    sqlc.narg(platform),
    sqlc.narg(min_osquery_version),
    @schedule_interval,
    sqlc.narg(created_by_user_id)
)
RETURNING
    id,
    name,
    description,
    query,
    platform,
    min_osquery_version,
    schedule_interval,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at;

-- name: UpdateSavedQuery :one
UPDATE queries
SET
    name = @name,
    description = @description,
    query = @query,
    platform = sqlc.narg(platform),
    min_osquery_version = sqlc.narg(min_osquery_version),
    schedule_interval = @schedule_interval,
    updated_at = now()
WHERE id = @id
RETURNING
    id,
    name,
    description,
    query,
    platform,
    min_osquery_version,
    schedule_interval,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at;

-- name: DeleteSavedQuery :one
DELETE FROM queries
WHERE id = @id
RETURNING id;

-- name: DeleteSavedQueries :many
DELETE FROM queries
WHERE id = ANY(@ids::bigint[])
RETURNING id;
