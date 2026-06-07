-- name: GetReportByID :one
SELECT
    id,
    name,
    description,
    query,
    min_osquery_version,
    schedule_interval,
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
    r.created_by_user_id,
    r.created_at,
    r.updated_at
FROM reports r
JOIN host_row h ON true
WHERE r.schedule_interval > 0
  AND EXISTS (
      SELECT 1
      FROM osquery_report_targets rt
      JOIN label_membership lm ON lm.label_id = rt.label_id AND lm.host_id = h.id
      WHERE rt.report_id = r.id
        AND rt.direction = 'include'
  )
  AND NOT EXISTS (
      SELECT 1
      FROM osquery_report_targets rt
      JOIN label_membership lm ON lm.label_id = rt.label_id AND lm.host_id = h.id
      WHERE rt.report_id = r.id
        AND rt.direction = 'exclude'
  )
ORDER BY r.id;

-- name: ListReportTargets :many
SELECT report_id, label_id, direction::text AS effect
FROM osquery_report_targets
WHERE report_id = ANY(@report_ids::bigint[])
ORDER BY
    report_id,
    CASE direction WHEN 'exclude' THEN 0 ELSE 1 END,
    position;

-- name: DeleteReportTargets :exec
DELETE FROM osquery_report_targets
WHERE report_id = @report_id;

-- name: InsertReportTargets :exec
INSERT INTO osquery_report_targets (report_id, label_id, direction, position)
SELECT @report_id, labels.label_id, effects.effect::target_direction, labels.ord - 1
FROM unnest(@label_ids::bigint[]) WITH ORDINALITY AS labels(label_id, ord)
JOIN unnest(@effects::text[]) WITH ORDINALITY AS effects(effect, ord) USING (ord);

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
