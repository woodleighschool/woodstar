-- name: UpsertSantaHostObservation :exec
INSERT INTO santa_hosts (
    host_id,
    machine_id,
    serial_number,
    santa_version,
    client_mode_reported,
    primary_user,
    primary_user_groups,
    sip_status,
    os_build,
    model_identifier,
    last_seen_at
)
VALUES (
    @host_id,
    @machine_id,
    @serial_number,
    @santa_version,
    @client_mode_reported::santa_client_mode,
    @primary_user,
    @primary_user_groups,
    @sip_status,
    @os_build,
    @model_identifier,
    COALESCE(sqlc.narg(last_seen_at)::timestamptz, now())
)
ON CONFLICT (host_id) DO UPDATE SET
    machine_id = EXCLUDED.machine_id,
    serial_number = EXCLUDED.serial_number,
    santa_version = EXCLUDED.santa_version,
    client_mode_reported = EXCLUDED.client_mode_reported,
    primary_user = EXCLUDED.primary_user,
    primary_user_groups = EXCLUDED.primary_user_groups,
    sip_status = EXCLUDED.sip_status,
    os_build = EXCLUDED.os_build,
    model_identifier = EXCLUDED.model_identifier,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = now();

-- name: CreateSantaConfiguration :one
INSERT INTO santa_configurations (
    name,
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

-- name: CreateSantaRule :one
INSERT INTO santa_rules (
    rule_type,
    identifier,
    name,
    custom_message,
    custom_url
)
VALUES (
    @rule_type::santa_rule_type,
    @identifier,
    @name,
    @custom_message,
    @custom_url
)
RETURNING *;

-- name: GetSantaRuleByID :one
SELECT *
FROM santa_rules
WHERE id = @id;

-- name: UpdateSantaRule :one
UPDATE santa_rules
SET
    rule_type = @rule_type::santa_rule_type,
    identifier = @identifier,
    name = @name,
    custom_message = @custom_message,
    custom_url = @custom_url,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteSantaRule :one
DELETE FROM santa_rules
WHERE id = @id
RETURNING id;

-- name: DeleteSantaRules :many
DELETE FROM santa_rules
WHERE id = ANY(@ids::bigint[])
RETURNING id;
