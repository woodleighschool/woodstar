-- name: GetCheckByID :one
SELECT
    id,
    name,
    description,
    query,
    platform,
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
    created_by_user_id
)
VALUES (
    @name,
    @description,
    @query,
    sqlc.narg(platform),
    sqlc.narg(created_by_user_id)
)
RETURNING
    id,
    name,
    description,
    query,
    platform,
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
    updated_at = now()
WHERE id = @id
RETURNING
    id,
    name,
    description,
    query,
    platform,
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
    SELECT
        id,
        lower(platform) AS platform
    FROM hosts h
    WHERE h.id = @host_id AND h.deleted_at IS NULL
)
SELECT
    c.id,
    c.name,
    c.description,
    c.query,
    c.platform,
    c.label_scope_mode,
    c.created_by_user_id,
    c.created_at,
    c.updated_at
FROM checks c
JOIN host_row h ON true
WHERE (
      c.platform IS NULL
      OR c.platform::text = h.platform
      OR (c.platform = 'darwin' AND h.platform = 'macos')
      OR (c.platform = 'linux' AND h.platform <> '' AND h.platform NOT IN ('darwin', 'macos', 'windows'))
  )
  AND (
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
        display_name,
        lower(platform) AS platform
    FROM hosts
    WHERE deleted_at IS NULL
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
      c.platform IS NULL
      OR c.platform::text = h.platform
      OR (c.platform = 'darwin' AND h.platform = 'macos')
      OR (c.platform = 'linux' AND h.platform <> '' AND h.platform NOT IN ('darwin', 'macos', 'windows'))
  )
  AND (
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
        display_name,
        lower(platform) AS platform
    FROM hosts h
    WHERE h.id = @host_id AND h.deleted_at IS NULL
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
      c.platform IS NULL
      OR c.platform::text = h.platform
      OR (c.platform = 'darwin' AND h.platform = 'macos')
      OR (c.platform = 'linux' AND h.platform <> '' AND h.platform NOT IN ('darwin', 'macos', 'windows'))
  )
  AND (
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
