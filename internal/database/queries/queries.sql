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

-- name: ListScheduledQueriesForHost :many
WITH host_row AS (
    SELECT
        id,
        lower(platform) AS platform
    FROM hosts h
    WHERE h.id = @host_id AND h.deleted_at IS NULL
)
SELECT
    q.id,
    q.name,
    q.description,
    q.query,
    q.platform,
    q.min_osquery_version,
    q.schedule_interval,
    q.label_scope_mode,
    q.created_by_user_id,
    q.created_at,
    q.updated_at
FROM queries q
JOIN host_row h ON true
WHERE q.schedule_interval > 0
  AND (
      q.platform IS NULL
      OR q.platform::text = h.platform
      OR (q.platform = 'darwin' AND h.platform = 'macos')
      OR (q.platform = 'linux' AND h.platform <> '' AND h.platform NOT IN ('darwin', 'macos', 'windows'))
  )
  AND (
      q.label_scope_mode = 'none'
      OR (
          q.label_scope_mode = 'include_any'
          AND EXISTS (
              SELECT 1
              FROM query_labels ql
              JOIN label_membership lm ON lm.label_id = ql.label_id AND lm.host_id = h.id
              WHERE ql.query_id = q.id
          )
      )
      OR (
          q.label_scope_mode = 'include_all'
          AND NOT EXISTS (
              SELECT 1
              FROM query_labels ql
              WHERE ql.query_id = q.id
                AND NOT EXISTS (
                    SELECT 1
                    FROM label_membership lm
                    WHERE lm.label_id = ql.label_id AND lm.host_id = h.id
                )
          )
      )
      OR (
          q.label_scope_mode = 'exclude_any'
          AND NOT EXISTS (
              SELECT 1
              FROM query_labels ql
              JOIN label_membership lm ON lm.label_id = ql.label_id AND lm.host_id = h.id
              WHERE ql.query_id = q.id
          )
      )
  )
ORDER BY q.id;
