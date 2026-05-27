-- name: GetReportByID :one
SELECT
    id,
    name,
    description,
    query,
    min_osquery_version,
    schedule_interval,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at
FROM reports
WHERE id = @id;

-- name: CreateReport :one
INSERT INTO reports (
    name,
    description,
    query,
    min_osquery_version,
    schedule_interval,
    created_by_user_id
)
VALUES (
    @name,
    @description,
    @query,
    sqlc.narg(min_osquery_version),
    @schedule_interval,
    sqlc.narg(created_by_user_id)
)
RETURNING
    id,
    name,
    description,
    query,
    min_osquery_version,
    schedule_interval,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at;

-- name: UpdateReport :one
UPDATE reports
SET
    name = @name,
    description = @description,
    query = @query,
    min_osquery_version = sqlc.narg(min_osquery_version),
    schedule_interval = @schedule_interval,
    updated_at = now()
WHERE id = @id
RETURNING
    id,
    name,
    description,
    query,
    min_osquery_version,
    schedule_interval,
    label_scope_mode,
    created_by_user_id,
    created_at,
    updated_at;

-- name: DeleteReport :one
DELETE FROM reports
WHERE id = @id
RETURNING id;

-- name: DeleteReports :many
DELETE FROM reports
WHERE id = ANY(@ids::bigint[])
RETURNING id;

-- name: ListScheduledReportsForHost :many
WITH host_row AS (
    SELECT id
    FROM hosts h
    WHERE h.id = @host_id
)
SELECT
    r.id,
    r.name,
    r.description,
    r.query,
    r.min_osquery_version,
    r.schedule_interval,
    r.label_scope_mode,
    r.created_by_user_id,
    r.created_at,
    r.updated_at
FROM reports r
JOIN host_row h ON true
WHERE r.schedule_interval > 0
  AND (
      r.label_scope_mode = 'none'
      OR (
          r.label_scope_mode = 'include_any'
          AND EXISTS (
              SELECT 1
              FROM report_labels rl
              JOIN label_membership lm ON lm.label_id = rl.label_id AND lm.host_id = h.id
              WHERE rl.report_id = r.id
          )
      )
      OR (
          r.label_scope_mode = 'include_all'
          AND NOT EXISTS (
              SELECT 1
              FROM report_labels rl
              WHERE rl.report_id = r.id
                AND NOT EXISTS (
                    SELECT 1
                    FROM label_membership lm
                    WHERE lm.label_id = rl.label_id AND lm.host_id = h.id
                )
          )
      )
      OR (
          r.label_scope_mode = 'exclude_any'
          AND NOT EXISTS (
              SELECT 1
              FROM report_labels rl
              JOIN label_membership lm ON lm.label_id = rl.label_id AND lm.host_id = h.id
              WHERE rl.report_id = r.id
          )
      )
  )
ORDER BY r.id;
