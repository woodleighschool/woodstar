-- name: ListLabels :many
SELECT
    l.id,
    l.name,
    l.description,
    l.query,
    l.kind,
    l.membership_type,
    l.platform,
    count(lm.host_id)::integer AS hosts_count,
    l.created_at,
    l.updated_at
FROM labels l
LEFT JOIN label_membership lm ON lm.label_id = l.id
WHERE
    (@q::text = '' OR l.name ILIKE '%' || @q::text || '%' OR l.description ILIKE '%' || @q::text || '%')
    AND (@kind::text = '' OR l.kind = @kind::text)
    AND (@membership_type::text = '' OR l.membership_type = @membership_type::text)
    AND (@platform::text = '' OR l.platform = @platform::text)
GROUP BY l.id
ORDER BY
    CASE WHEN @order_key::text = 'name' AND @order_direction::text = 'asc' THEN lower(l.name) END ASC,
    CASE WHEN @order_key::text = 'name' AND @order_direction::text = 'desc' THEN lower(l.name) END DESC,
    CASE WHEN @order_key::text = 'kind' AND @order_direction::text = 'asc' THEN l.kind END ASC,
    CASE WHEN @order_key::text = 'kind' AND @order_direction::text = 'desc' THEN l.kind END DESC,
    CASE WHEN @order_key::text = 'membership_type' AND @order_direction::text = 'asc' THEN l.membership_type END ASC,
    CASE WHEN @order_key::text = 'membership_type' AND @order_direction::text = 'desc' THEN l.membership_type END DESC,
    CASE WHEN @order_key::text = 'platform' AND @order_direction::text = 'asc' THEN l.platform END ASC NULLS LAST,
    CASE WHEN @order_key::text = 'platform' AND @order_direction::text = 'desc' THEN l.platform END DESC NULLS LAST,
    CASE WHEN @order_key::text = 'hosts_count' AND @order_direction::text = 'asc' THEN count(lm.host_id) END ASC,
    CASE WHEN @order_key::text = 'hosts_count' AND @order_direction::text = 'desc' THEN count(lm.host_id) END DESC,
    CASE WHEN @order_key::text = 'updated_at' AND @order_direction::text = 'asc' THEN l.updated_at END ASC,
    CASE WHEN @order_key::text = 'updated_at' AND @order_direction::text = 'desc' THEN l.updated_at END DESC,
    lower(l.name),
    l.id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountLabels :one
SELECT count(*)::integer
FROM labels l
WHERE
    (@q::text = '' OR l.name ILIKE '%' || @q::text || '%' OR l.description ILIKE '%' || @q::text || '%')
    AND (@kind::text = '' OR l.kind = @kind::text)
    AND (@membership_type::text = '' OR l.membership_type = @membership_type::text)
    AND (@platform::text = '' OR l.platform = @platform::text);

-- name: GetLabelByID :one
SELECT
    l.id,
    l.name,
    l.description,
    l.query,
    l.kind,
    l.membership_type,
    l.platform,
    count(lm.host_id)::integer AS hosts_count,
    l.created_at,
    l.updated_at
FROM labels l
LEFT JOIN label_membership lm ON lm.label_id = l.id
WHERE l.id = @id
GROUP BY l.id;

-- name: CreateLabel :one
INSERT INTO labels (
    name,
    description,
    query,
    kind,
    membership_type,
    platform
)
VALUES (
    @name,
    @description,
    sqlc.narg(query),
    @kind,
    @membership_type,
    sqlc.narg(platform)
)
RETURNING
    id,
    name,
    description,
    query,
    kind,
    membership_type,
    platform,
    0::integer AS hosts_count,
    created_at,
    updated_at;

-- name: UpdateLabel :one
UPDATE labels
SET
    name = @name,
    description = @description,
    query = sqlc.narg(query),
    kind = @kind,
    membership_type = @membership_type,
    platform = sqlc.narg(platform),
    updated_at = now()
WHERE id = @id AND kind = 'custom'
RETURNING
    id,
    name,
    description,
    query,
    kind,
    membership_type,
    platform,
    (
        SELECT count(*)::integer
        FROM label_membership lm
        WHERE lm.label_id = labels.id
    ) AS hosts_count,
    created_at,
    updated_at;

-- name: DeleteCustomLabel :one
DELETE FROM labels
WHERE id = @id AND kind = 'custom'
RETURNING id;

-- name: ListApplicableDynamicLabels :many
SELECT
    id,
    name,
    kind,
    membership_type,
    query,
    platform
FROM labels
WHERE
    membership_type = 'dynamic'
    AND (platform IS NULL OR platform = '' OR platform = @platform::text)
ORDER BY id;

-- name: UpsertLabelMembership :exec
INSERT INTO label_membership (
    label_id,
    host_id,
    created_at,
    updated_at
)
VALUES (
    @label_id,
    @host_id,
    now(),
    now()
)
ON CONFLICT (label_id, host_id) DO UPDATE SET
    updated_at = now();

-- name: DeleteLabelMembership :exec
DELETE FROM label_membership
WHERE label_id = @label_id AND host_id = @host_id;

-- name: ListLabelsForHost :many
SELECT
    l.id,
    l.name,
    l.description,
    l.query,
    l.kind,
    l.membership_type,
    l.platform,
    count(lm_all.host_id)::integer AS hosts_count,
    l.created_at,
    l.updated_at
FROM labels l
JOIN label_membership lm_host ON lm_host.label_id = l.id AND lm_host.host_id = @host_id
LEFT JOIN label_membership lm_all ON lm_all.label_id = l.id
GROUP BY l.id
ORDER BY lower(l.name), l.id;

-- name: ListApplicableDynamicLabelIDs :many
SELECT id
FROM labels
WHERE
    id = ANY(@ids::bigint[])
    AND membership_type = 'dynamic'
    AND (platform IS NULL OR platform = '' OR platform = @platform::text)
ORDER BY id;

-- name: MarkHostLabelsFresh :exec
UPDATE hosts
SET label_updated_at = now(), updated_at = now()
WHERE id = @host_id AND deleted_at IS NULL;
