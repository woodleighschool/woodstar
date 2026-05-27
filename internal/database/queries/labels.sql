-- name: GetLabelByID :one
SELECT
    sqlc.embed(l),
    count(lm.host_id)::integer AS hosts_count
FROM labels l
LEFT JOIN label_membership lm ON lm.label_id = l.id
WHERE l.id = @id
GROUP BY l.id;

-- name: CreateLabel :one
INSERT INTO labels (
    name,
    description,
    query,
    criteria,
    label_type,
    label_membership_type
)
VALUES (
    @name,
    @description,
    sqlc.narg(query),
    sqlc.narg(criteria),
    @label_type,
    @label_membership_type
)
RETURNING *;

-- name: UpdateLabel :one
UPDATE labels
SET
    name = @name,
    description = @description,
    query = sqlc.narg(query),
    criteria = sqlc.narg(criteria),
    label_membership_type = @label_membership_type,
    updated_at = now()
WHERE id = @id AND label_type = 'regular'
RETURNING *;

-- name: DeleteRegularLabel :one
DELETE FROM labels
WHERE id = @id AND label_type = 'regular'
RETURNING id;

-- name: ListApplicableDynamicLabels :many
SELECT *
FROM labels
WHERE
    label_membership_type = 'dynamic'
ORDER BY id;

-- name: ListApplicableDynamicLabelIDs :many
SELECT id
FROM labels
WHERE
    id = ANY(@ids::bigint[])
    AND label_membership_type = 'dynamic'
ORDER BY id;

-- name: UpsertLabelMembership :exec
INSERT INTO label_membership (label_id, host_id)
VALUES (@label_id, @host_id)
ON CONFLICT (label_id, host_id) DO UPDATE SET
    updated_at = now();

-- name: DeleteLabelMembership :exec
DELETE FROM label_membership
WHERE label_id = @label_id AND host_id = @host_id;

-- name: DeleteLabelMembershipsForLabel :exec
DELETE FROM label_membership
WHERE label_id = @label_id;

-- name: InsertLabelMemberships :exec
INSERT INTO label_membership (label_id, host_id)
SELECT @label_id, unnest(@host_ids::bigint[])
ON CONFLICT (label_id, host_id) DO UPDATE SET
    updated_at = now();

-- name: ListManualLabelHostIDs :many
SELECT host_id
FROM label_membership lm
JOIN labels l ON l.id = lm.label_id
WHERE lm.label_id = @label_id AND l.label_membership_type = 'manual'
ORDER BY lm.host_id;

-- name: ListLabelsForHost :many
SELECT
    sqlc.embed(l),
    count(lm_all.host_id)::integer AS hosts_count
FROM labels l
JOIN label_membership lm_host ON lm_host.label_id = l.id AND lm_host.host_id = @host_id
LEFT JOIN label_membership lm_all ON lm_all.label_id = l.id
GROUP BY l.id
ORDER BY lower(l.name), l.id;

-- name: MarkHostLabelsFresh :exec
UPDATE hosts
SET label_updated_at = now(), updated_at = now()
WHERE id = @host_id;
