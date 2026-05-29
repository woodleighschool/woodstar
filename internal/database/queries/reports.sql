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

-- name: ListReportScopes :many
SELECT id, label_scope_mode
FROM reports
WHERE id = ANY(@report_ids::bigint[]);

-- name: ListReportLabelIDs :many
SELECT report_id, label_id
FROM report_labels
WHERE report_id = ANY(@report_ids::bigint[])
ORDER BY report_id, label_id;

-- name: SetReportScopeMode :exec
UPDATE reports
SET label_scope_mode = @label_scope_mode
WHERE id = @id;

-- name: DeleteReportLabels :exec
DELETE FROM report_labels
WHERE report_id = @report_id;

-- name: InsertReportLabels :exec
INSERT INTO report_labels (report_id, label_id)
SELECT @report_id, unnest(@label_ids::bigint[]);

-- name: DeleteReportResults :exec
DELETE FROM report_results
WHERE report_id = @report_id AND host_id = @host_id;

-- name: ListReportResults :many
SELECT rr.report_id, r.name, rr.host_id, h.display_name, rr.data, rr.last_fetched
FROM report_results rr
JOIN reports r ON r.id = rr.report_id
JOIN hosts h ON h.id = rr.host_id
WHERE rr.report_id = @report_id AND rr.data IS NOT NULL
ORDER BY rr.last_fetched DESC, rr.host_id, rr.id;

-- name: ListHostReportResults :many
SELECT rr.report_id, r.name, rr.host_id, h.display_name, rr.data, rr.last_fetched
FROM report_results rr
JOIN reports r ON r.id = rr.report_id
JOIN hosts h ON h.id = rr.host_id
WHERE rr.report_id = @report_id AND rr.host_id = @host_id
ORDER BY rr.last_fetched DESC, rr.id;

-- name: ListHostReportStates :many
WITH requested AS (
    SELECT unnest(@report_ids::bigint[])::bigint AS report_id
),
latest_fetch AS (
    SELECT DISTINCT ON (report_id) report_id, last_fetched
    FROM report_results rr
    WHERE rr.host_id = @state_host_id AND rr.report_id = ANY(@report_ids::bigint[])
    ORDER BY report_id, last_fetched DESC, id DESC
),
result_counts AS (
    SELECT report_id, count(*)::integer AS host_result_count
    FROM report_results rr
    WHERE rr.host_id = @state_host_id AND rr.report_id = ANY(@report_ids::bigint[]) AND rr.data IS NOT NULL
    GROUP BY report_id
),
latest_data AS (
    SELECT DISTINCT ON (report_id) report_id, data
    FROM report_results rr
    WHERE rr.host_id = @state_host_id AND rr.report_id = ANY(@report_ids::bigint[]) AND rr.data IS NOT NULL
    ORDER BY report_id, last_fetched DESC, id DESC
)
SELECT
    req.report_id,
    lf.last_fetched,
    coalesce(rc.host_result_count, 0)::integer AS host_result_count,
    ld.data
FROM requested req
LEFT JOIN latest_fetch lf ON lf.report_id = req.report_id
LEFT JOIN result_counts rc ON rc.report_id = req.report_id
LEFT JOIN latest_data ld ON ld.report_id = req.report_id;
