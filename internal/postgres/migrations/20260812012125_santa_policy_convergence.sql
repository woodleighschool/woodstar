-- +goose Up

CREATE TYPE santa_sync_type AS ENUM ('normal', 'clean', 'clean_all');
CREATE TYPE santa_remount_flag AS ENUM (
    'rdonly',
    'noexec',
    'nosuid',
    'nobrowse',
    'noowners',
    'nodev',
    '-j',
    'async'
);

ALTER TABLE santa_configurations
    DROP CONSTRAINT santa_configurations_check,
    DROP CONSTRAINT santa_configurations_check1,
    ALTER COLUMN allowed_path_regex DROP NOT NULL,
    ALTER COLUMN blocked_path_regex DROP NOT NULL,
    ALTER COLUMN event_detail_url DROP NOT NULL,
    ALTER COLUMN event_detail_text DROP NOT NULL,
    ADD CONSTRAINT santa_configurations_client_mode_check CHECK (
        client_mode <> 'unknown'
    );

UPDATE santa_configurations
SET
    allowed_path_regex = NULLIF(btrim(allowed_path_regex), ''),
    blocked_path_regex = NULLIF(btrim(blocked_path_regex), ''),
    event_detail_url = NULLIF(btrim(event_detail_url), ''),
    event_detail_text = NULLIF(btrim(event_detail_text), '');

ALTER TABLE santa_configurations
    ADD CONSTRAINT santa_configurations_allowed_path_regex_check CHECK (
        allowed_path_regex IS NULL OR NULLIF(btrim(allowed_path_regex), '') IS NOT NULL
    ),
    ADD CONSTRAINT santa_configurations_blocked_path_regex_check CHECK (
        blocked_path_regex IS NULL OR NULLIF(btrim(blocked_path_regex), '') IS NOT NULL
    ),
    ADD CONSTRAINT santa_configurations_event_detail_url_check CHECK (
        event_detail_url IS NULL OR NULLIF(btrim(event_detail_url), '') IS NOT NULL
    ),
    ADD CONSTRAINT santa_configurations_event_detail_text_check CHECK (
        event_detail_text IS NULL OR NULLIF(btrim(event_detail_text), '') IS NOT NULL
    );

WITH normalized AS (
    SELECT
        id,
        ARRAY(
            SELECT allowed.flag
            FROM unnest(ARRAY[
                'rdonly', 'noexec', 'nosuid', 'nobrowse',
                'noowners', 'nodev', '-j', 'async'
            ]::TEXT[]) WITH ORDINALITY AS allowed(flag, position)
            WHERE allowed.flag = ANY(COALESCE(removable_media_remount_flags, ARRAY[]::TEXT[]))
            ORDER BY allowed.position
        ) AS removable_flags,
        ARRAY(
            SELECT allowed.flag
            FROM unnest(ARRAY[
                'rdonly', 'noexec', 'nosuid', 'nobrowse',
                'noowners', 'nodev', '-j', 'async'
            ]::TEXT[]) WITH ORDINALITY AS allowed(flag, position)
            WHERE allowed.flag = ANY(
                COALESCE(encrypted_removable_media_remount_flags, ARRAY[]::TEXT[])
            )
            ORDER BY allowed.position
        ) AS encrypted_flags
    FROM santa_configurations
)
UPDATE santa_configurations c
SET
    removable_media_action = CASE
        WHEN c.removable_media_action = 'remount' AND cardinality(n.removable_flags) = 0 THEN NULL
        ELSE c.removable_media_action
    END,
    removable_media_remount_flags = CASE
        WHEN c.removable_media_action = 'remount' AND cardinality(n.removable_flags) > 0
            THEN n.removable_flags
        ELSE NULL
    END,
    encrypted_removable_media_action = CASE
        WHEN c.encrypted_removable_media_action = 'remount'
            AND cardinality(n.encrypted_flags) = 0 THEN NULL
        ELSE c.encrypted_removable_media_action
    END,
    encrypted_removable_media_remount_flags = CASE
        WHEN c.encrypted_removable_media_action = 'remount'
            AND cardinality(n.encrypted_flags) > 0 THEN n.encrypted_flags
        ELSE NULL
    END
FROM normalized n
WHERE n.id = c.id;

ALTER TABLE santa_configurations
    ALTER COLUMN removable_media_remount_flags TYPE santa_remount_flag[]
        USING removable_media_remount_flags::santa_remount_flag[],
    ALTER COLUMN encrypted_removable_media_remount_flags TYPE santa_remount_flag[]
        USING encrypted_removable_media_remount_flags::santa_remount_flag[];

ALTER TABLE santa_configurations
    ADD CONSTRAINT santa_configurations_removable_media_policy_check CHECK (
        CASE
            WHEN removable_media_action IS NULL THEN removable_media_remount_flags IS NULL
            WHEN removable_media_action = 'remount' THEN
                COALESCE(cardinality(removable_media_remount_flags), 0) > 0
                AND array_ndims(removable_media_remount_flags) = 1
                AND array_position(removable_media_remount_flags, NULL) IS NULL
                AND cardinality(removable_media_remount_flags) =
                    (('rdonly' = ANY(removable_media_remount_flags))::INT
                    + ('noexec' = ANY(removable_media_remount_flags))::INT
                    + ('nosuid' = ANY(removable_media_remount_flags))::INT
                    + ('nobrowse' = ANY(removable_media_remount_flags))::INT
                    + ('noowners' = ANY(removable_media_remount_flags))::INT
                    + ('nodev' = ANY(removable_media_remount_flags))::INT
                    + ('-j' = ANY(removable_media_remount_flags))::INT
                    + ('async' = ANY(removable_media_remount_flags))::INT)
            ELSE removable_media_remount_flags IS NULL
        END
    ),
    ADD CONSTRAINT santa_configurations_encrypted_removable_media_policy_check CHECK (
        CASE
            WHEN encrypted_removable_media_action IS NULL
                THEN encrypted_removable_media_remount_flags IS NULL
            WHEN encrypted_removable_media_action = 'remount' THEN
                COALESCE(cardinality(encrypted_removable_media_remount_flags), 0) > 0
                AND array_ndims(encrypted_removable_media_remount_flags) = 1
                AND array_position(encrypted_removable_media_remount_flags, NULL) IS NULL
                AND cardinality(encrypted_removable_media_remount_flags) =
                    (('rdonly' = ANY(encrypted_removable_media_remount_flags))::INT
                    + ('noexec' = ANY(encrypted_removable_media_remount_flags))::INT
                    + ('nosuid' = ANY(encrypted_removable_media_remount_flags))::INT
                    + ('nobrowse' = ANY(encrypted_removable_media_remount_flags))::INT
                    + ('noowners' = ANY(encrypted_removable_media_remount_flags))::INT
                    + ('nodev' = ANY(encrypted_removable_media_remount_flags))::INT
                    + ('-j' = ANY(encrypted_removable_media_remount_flags))::INT
                    + ('async' = ANY(encrypted_removable_media_remount_flags))::INT)
            ELSE encrypted_removable_media_remount_flags IS NULL
        END
    );

ALTER TABLE santa_sync_state
    DROP CONSTRAINT santa_sync_state_preflight_rules_hash_check,
    DROP CONSTRAINT santa_sync_state_confirmed_rules_hash_check,
    ALTER COLUMN preflight_rules_hash DROP NOT NULL,
    ALTER COLUMN preflight_rules_hash DROP DEFAULT,
    ALTER COLUMN confirmed_rules_hash DROP NOT NULL,
    ALTER COLUMN confirmed_rules_hash DROP DEFAULT,
    ADD COLUMN pending_sync_type santa_sync_type,
    ADD COLUMN pending_policy_digest TEXT,
    ADD COLUMN applied_policy_digest TEXT;

UPDATE santa_sync_state
SET
    preflight_rules_hash = NULL,
    confirmed_rules_hash = NULLIF(confirmed_rules_hash, '');

ALTER TABLE santa_sync_state
    DROP COLUMN pending_full_sync,
    DROP COLUMN pending_payload_rule_count,
    DROP COLUMN pending_preflight_at,
    DROP COLUMN desired_binary_rule_count,
    DROP COLUMN desired_certificate_rule_count,
    DROP COLUMN desired_teamid_rule_count,
    DROP COLUMN desired_signingid_rule_count,
    DROP COLUMN desired_cdhash_rule_count,
    DROP COLUMN desired_compiler_rule_count,
    DROP COLUMN binary_rule_count,
    DROP COLUMN certificate_rule_count,
    DROP COLUMN teamid_rule_count,
    DROP COLUMN signingid_rule_count,
    DROP COLUMN cdhash_rule_count,
    DROP COLUMN compiler_rule_count,
    DROP COLUMN transitive_rule_count,
    DROP COLUMN rules_received,
    DROP COLUMN rules_processed,
    DROP COLUMN last_rule_sync_attempt_at,
    DROP COLUMN last_rule_sync_success_at,
    DROP COLUMN last_reported_counts_match_at,
    ADD CONSTRAINT santa_sync_state_preflight_rules_hash_check CHECK (
        preflight_rules_hash IS NULL OR preflight_rules_hash ~ '^[0-9a-f]{32}$'
    ),
    ADD CONSTRAINT santa_sync_state_confirmed_rules_hash_check CHECK (
        confirmed_rules_hash IS NULL OR confirmed_rules_hash ~ '^[0-9a-f]{32}$'
    ),
    ADD CONSTRAINT santa_sync_state_pending_policy_digest_check CHECK (
        pending_policy_digest IS NULL OR pending_policy_digest ~ '^[0-9a-f]{64}$'
    ),
    ADD CONSTRAINT santa_sync_state_applied_policy_digest_check CHECK (
        applied_policy_digest IS NULL OR applied_policy_digest ~ '^[0-9a-f]{64}$'
    ),
    ADD CONSTRAINT santa_sync_state_pending_policy_check CHECK (
        (
            pending_sync_type IS NULL
            AND pending_policy_digest IS NULL
            AND preflight_rules_hash IS NULL
        )
        OR (
            pending_sync_type IS NOT NULL
            AND pending_policy_digest IS NOT NULL
            AND preflight_rules_hash IS NOT NULL
        )
    );

ALTER TABLE santa_sync_targets
    DROP COLUMN payload_hash;
