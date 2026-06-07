-- name: CreateSantaConfiguration :one
INSERT INTO santa_configurations (
    name,
    description,
    position,
    client_mode,
    enable_bundles,
    enable_transitive_rules,
    enable_all_event_upload,
    full_sync_interval_seconds,
    batch_size,
    allowed_path_regex,
    blocked_path_regex,
    removable_media_action,
    removable_media_remount_flags,
    encrypted_removable_media_action,
    encrypted_removable_media_remount_flags,
    event_detail_url,
    event_detail_text
)
VALUES (
    @name,
    @description,
    (SELECT COALESCE(MAX(position) + 1, 0) FROM santa_configurations),
    @client_mode::santa_client_mode,
    @enable_bundles,
    @enable_transitive_rules,
    @enable_all_event_upload,
    @full_sync_interval_seconds::integer,
    @batch_size::integer,
    @allowed_path_regex,
    @blocked_path_regex,
    sqlc.narg(removable_media_action)::santa_removable_media_action,
    sqlc.narg(removable_media_remount_flags)::text[],
    sqlc.narg(encrypted_removable_media_action)::santa_removable_media_action,
    sqlc.narg(encrypted_removable_media_remount_flags)::text[],
    @event_detail_url,
    @event_detail_text
)
RETURNING *;

-- name: GetSantaConfigurationByID :one
SELECT *
FROM santa_configurations
WHERE id = @id;

-- name: UpdateSantaConfiguration :one
UPDATE santa_configurations
SET
    name = @name,
    description = @description,
    client_mode = @client_mode::santa_client_mode,
    enable_bundles = @enable_bundles,
    enable_transitive_rules = @enable_transitive_rules,
    enable_all_event_upload = @enable_all_event_upload,
    full_sync_interval_seconds = @full_sync_interval_seconds::integer,
    batch_size = @batch_size::integer,
    allowed_path_regex = @allowed_path_regex,
    blocked_path_regex = @blocked_path_regex,
    removable_media_action = sqlc.narg(removable_media_action)::santa_removable_media_action,
    removable_media_remount_flags = sqlc.narg(removable_media_remount_flags)::text[],
    encrypted_removable_media_action = sqlc.narg(encrypted_removable_media_action)::santa_removable_media_action,
    encrypted_removable_media_remount_flags = sqlc.narg(encrypted_removable_media_remount_flags)::text[],
    event_detail_url = @event_detail_url,
    event_detail_text = @event_detail_text,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteSantaConfiguration :one
DELETE FROM santa_configurations
WHERE id = @id
RETURNING id;

-- name: DeleteSantaConfigurations :many
DELETE FROM santa_configurations
WHERE id = ANY(@ids::bigint[])
RETURNING id;

-- name: ListSantaConfigurationIDsByPosition :many
SELECT id
FROM santa_configurations
ORDER BY position, id;

-- name: SetSantaConfigurationPositions :exec
UPDATE santa_configurations c
SET position = -ordered.position
FROM unnest(@ordered_ids::bigint[]) WITH ORDINALITY AS ordered(id, position)
WHERE c.id = ordered.id;

-- name: NormalizeSantaConfigurationPositions :exec
UPDATE santa_configurations
SET position = -position - 1;

-- name: ResolveSantaConfigurationForHost :one
SELECT
    sqlc.embed(c),
    l.id AS label_id,
    l.name AS label_name
FROM santa_configurations c
JOIN LATERAL (
    SELECT
        include_label.id,
        include_label.name
    FROM santa_configuration_targets t
    JOIN label_membership lm ON lm.label_id = t.label_id AND lm.host_id = @host_id
    JOIN labels include_label ON include_label.id = t.label_id
    WHERE t.configuration_id = c.id
      AND t.direction = 'include'
    ORDER BY t.position
    LIMIT 1
) l ON true
WHERE NOT EXISTS (
    SELECT 1
    FROM santa_configuration_targets t
    JOIN label_membership lm ON lm.label_id = t.label_id AND lm.host_id = @host_id
    WHERE t.configuration_id = c.id
      AND t.direction = 'exclude'
)
ORDER BY c.position, c.id
LIMIT 1;

-- name: DeleteSantaConfigurationTargets :exec
DELETE FROM santa_configuration_targets
WHERE configuration_id = @configuration_id;

-- name: InsertSantaConfigurationTargets :exec
INSERT INTO santa_configuration_targets (configuration_id, label_id, direction, position)
SELECT @configuration_id, labels.label_id, effects.effect::target_direction, labels.ord - 1
FROM unnest(@label_ids::bigint[]) WITH ORDINALITY AS labels(label_id, ord)
JOIN unnest(@effects::text[]) WITH ORDINALITY AS effects(effect, ord) USING (ord);

-- name: ListSantaConfigurationTargets :many
SELECT configuration_id, label_id, direction::text AS effect
FROM santa_configuration_targets
WHERE configuration_id = ANY(@configuration_ids::bigint[])
ORDER BY configuration_id, direction, position;
