-- name: GetCheckByID :one
SELECT
    id,
    name,
    description,
    query,
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
    created_by_user_id
)
VALUES (
    @name,
    @description,
    @query,
    sqlc.narg(created_by_user_id)
)
RETURNING
    id,
    name,
    description,
    query,
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
    updated_at = now()
WHERE id = @id
RETURNING
    id,
    name,
    description,
    query,
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

-- name: ListApplicableChecksForHost :many
WITH host_row AS (
    SELECT id
    FROM hosts h
    WHERE h.id = @host_id
)
SELECT
    c.id,
    c.name,
    c.description,
    c.query,
    c.label_scope_mode,
    c.created_by_user_id,
    c.created_at,
    c.updated_at
FROM checks c
JOIN host_row h ON true
WHERE (
      c.label_scope_mode = 'none'
      OR (
          c.label_scope_mode = 'include_any'
          AND EXISTS (
              SELECT 1
              FROM check_labels cl
              JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = h.id
              WHERE cl.check_id = c.id
          )
      )
      OR (
          c.label_scope_mode = 'include_all'
          AND NOT EXISTS (
              SELECT 1
              FROM check_labels cl
              WHERE cl.check_id = c.id
                AND NOT EXISTS (
                    SELECT 1
                    FROM label_membership lm
                    WHERE lm.label_id = cl.label_id AND lm.host_id = h.id
                )
          )
      )
      OR (
          c.label_scope_mode = 'exclude_any'
          AND NOT EXISTS (
              SELECT 1
              FROM check_labels cl
              JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = h.id
              WHERE cl.check_id = c.id
          )
      )
  )
ORDER BY c.id;

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
WITH check_row AS (
    SELECT *
    FROM checks c
    WHERE c.id = @check_id
),
host_rows AS (
    SELECT
        id,
        display_name
    FROM hosts
)
SELECT
    c.id AS check_id,
    c.name AS check_name,
    h.id AS host_id,
    h.display_name AS host_name,
    m.passes,
    m.updated_at
FROM check_row c
JOIN host_rows h ON true
LEFT JOIN check_membership m ON m.host_id = h.id AND m.check_id = c.id
WHERE (
      c.label_scope_mode = 'none'
      OR (
          c.label_scope_mode = 'include_any'
          AND EXISTS (
              SELECT 1
              FROM check_labels cl
              JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = h.id
              WHERE cl.check_id = c.id
          )
      )
      OR (
          c.label_scope_mode = 'include_all'
          AND NOT EXISTS (
              SELECT 1
              FROM check_labels cl
              WHERE cl.check_id = c.id
                AND NOT EXISTS (
                    SELECT 1
                    FROM label_membership lm
                    WHERE lm.label_id = cl.label_id AND lm.host_id = h.id
                )
          )
      )
      OR (
          c.label_scope_mode = 'exclude_any'
          AND NOT EXISTS (
              SELECT 1
              FROM check_labels cl
              JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = h.id
              WHERE cl.check_id = c.id
          )
      )
  )
ORDER BY
    CASE
        WHEN m.passes IS FALSE THEN 0
        WHEN m.passes IS NULL THEN 1
        ELSE 2
    END,
    lower(h.display_name),
    h.id;

-- name: ListHostCheckStatusesForHost :many
WITH host_row AS (
    SELECT
        id,
        display_name
    FROM hosts h
    WHERE h.id = @host_id
)
SELECT
    c.id AS check_id,
    c.name AS check_name,
    h.id AS host_id,
    h.display_name AS host_name,
    m.passes,
    m.updated_at
FROM checks c
JOIN host_row h ON true
LEFT JOIN check_membership m ON m.host_id = h.id AND m.check_id = c.id
WHERE (
      c.label_scope_mode = 'none'
      OR (
          c.label_scope_mode = 'include_any'
          AND EXISTS (
              SELECT 1
              FROM check_labels cl
              JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = h.id
              WHERE cl.check_id = c.id
          )
      )
      OR (
          c.label_scope_mode = 'include_all'
          AND NOT EXISTS (
              SELECT 1
              FROM check_labels cl
              WHERE cl.check_id = c.id
                AND NOT EXISTS (
                    SELECT 1
                    FROM label_membership lm
                    WHERE lm.label_id = cl.label_id AND lm.host_id = h.id
                )
          )
      )
      OR (
          c.label_scope_mode = 'exclude_any'
          AND NOT EXISTS (
              SELECT 1
              FROM check_labels cl
              JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = h.id
              WHERE cl.check_id = c.id
          )
      )
  )
ORDER BY
    CASE
        WHEN m.passes IS FALSE THEN 0
        WHEN m.passes IS NULL THEN 1
        ELSE 2
    END,
    lower(c.name),
    c.id;

-- name: ListCheckScopes :many
SELECT id, label_scope_mode
FROM checks
WHERE id = ANY(@check_ids::bigint[]);

-- name: ListCheckLabelIDs :many
SELECT check_id, label_id
FROM check_labels
WHERE check_id = ANY(@check_ids::bigint[])
ORDER BY check_id, label_id;

-- name: SetCheckScopeMode :exec
UPDATE checks
SET label_scope_mode = @label_scope_mode
WHERE id = @id;

-- name: DeleteCheckLabels :exec
DELETE FROM check_labels
WHERE check_id = @check_id;

-- name: InsertCheckLabels :exec
INSERT INTO check_labels (check_id, label_id)
SELECT @check_id, unnest(@label_ids::bigint[]);

-- name: ListCheckCounts :many
SELECT
    check_id,
    COUNT(*) FILTER (WHERE passes IS TRUE)::integer AS passing_host_count,
    COUNT(*) FILTER (WHERE passes IS FALSE)::integer AS failing_host_count
FROM check_membership
WHERE check_id = ANY(@check_ids::bigint[])
GROUP BY check_id;
