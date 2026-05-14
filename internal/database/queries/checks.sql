-- name: GetCheckByID :one
SELECT
    id,
    name,
    description,
    query,
    platform,
    min_osquery_version,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at
FROM checks
WHERE id = @id;

-- name: CreateCheck :one
INSERT INTO checks (
    name,
    description,
    query,
    platform,
    min_osquery_version,
    created_by_user_id
)
VALUES (
    @name,
    @description,
    @query,
    sqlc.narg(platform),
    sqlc.narg(min_osquery_version),
    sqlc.narg(created_by_user_id)
)
RETURNING
    id,
    name,
    description,
    query,
    platform,
    min_osquery_version,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at;

-- name: UpdateCheck :one
UPDATE checks
SET
    name = @name,
    description = @description,
    query = @query,
    platform = sqlc.narg(platform),
    min_osquery_version = sqlc.narg(min_osquery_version),
    updated_at = now()
WHERE id = @id
RETURNING
    id,
    name,
    description,
    query,
    platform,
    min_osquery_version,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at;

-- name: DeleteCheck :one
DELETE FROM checks
WHERE id = @id
RETURNING id;

-- name: DeleteChecks :many
DELETE FROM checks
WHERE id = ANY(@ids::bigint[])
RETURNING id;

-- name: UpsertCheckMembership :exec
INSERT INTO check_membership (
    check_id,
    host_id,
    passes,
    updated_at
)
VALUES (
    @check_id,
    @host_id,
    sqlc.narg(passes),
    now()
)
ON CONFLICT (check_id, host_id) DO UPDATE SET
    passes = EXCLUDED.passes,
    updated_at = now();

-- name: ListCheckHostStatuses :many
SELECT
    c.id AS check_id,
    c.name AS check_name,
    h.id AS host_id,
    h.display_name AS host_name,
    m.passes,
    m.updated_at
FROM checks c
CROSS JOIN hosts h
LEFT JOIN check_membership m ON m.host_id = h.id AND m.check_id = c.id
WHERE c.id = @check_id AND h.deleted_at IS NULL
ORDER BY
    CASE
        WHEN m.passes IS FALSE THEN 0
        WHEN m.passes IS NULL THEN 1
        ELSE 2
    END,
    lower(h.display_name),
    h.id;

-- name: ListHostCheckStatuses :many
SELECT
    c.id AS check_id,
    c.name AS check_name,
    h.id AS host_id,
    h.display_name AS host_name,
    m.passes,
    m.updated_at
FROM checks c
JOIN hosts h ON h.id = @host_id AND h.deleted_at IS NULL
LEFT JOIN check_membership m ON m.host_id = h.id AND m.check_id = c.id
WHERE c.id = ANY(@check_ids::bigint[])
ORDER BY
    CASE
        WHEN m.passes IS FALSE THEN 0
        WHEN m.passes IS NULL THEN 1
        ELSE 2
    END,
    lower(c.name),
    c.id;
