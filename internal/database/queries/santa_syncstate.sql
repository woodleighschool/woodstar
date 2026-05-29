-- name: SantaSyncStateExists :one
SELECT EXISTS (
    SELECT 1
    FROM santa_sync_state
    WHERE host_id = @host_id
);

-- name: UpsertSantaSyncPreflight :exec
INSERT INTO santa_sync_state (
    host_id,
    client_rules_hash,
    pending_full_sync,
    pending_payload_rule_count,
    pending_preflight_at,
    desired_binary_rule_count,
    desired_certificate_rule_count,
    desired_teamid_rule_count,
    desired_signingid_rule_count,
    desired_cdhash_rule_count,
    binary_rule_count,
    certificate_rule_count,
    teamid_rule_count,
    signingid_rule_count,
    cdhash_rule_count,
    last_rule_sync_attempt_at,
    last_reported_counts_match_at,
    updated_at
)
VALUES (
    @host_id,
    @client_rules_hash,
    @pending_full_sync,
    @pending_payload_rule_count,
    now(),
    @desired_binary_rule_count,
    @desired_certificate_rule_count,
    @desired_teamid_rule_count,
    @desired_signingid_rule_count,
    @desired_cdhash_rule_count,
    @binary_rule_count,
    @certificate_rule_count,
    @teamid_rule_count,
    @signingid_rule_count,
    @cdhash_rule_count,
    now(),
    CASE WHEN @counts_match::boolean THEN now() ELSE NULL END,
    now()
)
ON CONFLICT (host_id) DO UPDATE SET
    client_rules_hash = EXCLUDED.client_rules_hash,
    pending_full_sync = EXCLUDED.pending_full_sync,
    pending_payload_rule_count = EXCLUDED.pending_payload_rule_count,
    pending_preflight_at = EXCLUDED.pending_preflight_at,
    desired_binary_rule_count = EXCLUDED.desired_binary_rule_count,
    desired_certificate_rule_count = EXCLUDED.desired_certificate_rule_count,
    desired_teamid_rule_count = EXCLUDED.desired_teamid_rule_count,
    desired_signingid_rule_count = EXCLUDED.desired_signingid_rule_count,
    desired_cdhash_rule_count = EXCLUDED.desired_cdhash_rule_count,
    binary_rule_count = EXCLUDED.binary_rule_count,
    certificate_rule_count = EXCLUDED.certificate_rule_count,
    teamid_rule_count = EXCLUDED.teamid_rule_count,
    signingid_rule_count = EXCLUDED.signingid_rule_count,
    cdhash_rule_count = EXCLUDED.cdhash_rule_count,
    last_rule_sync_attempt_at = EXCLUDED.last_rule_sync_attempt_at,
    last_reported_counts_match_at = CASE
        WHEN @counts_match::boolean THEN EXCLUDED.last_reported_counts_match_at
        ELSE santa_sync_state.last_reported_counts_match_at
    END,
    updated_at = now();

-- name: DeleteSantaSyncTargetsByPhase :exec
DELETE FROM santa_sync_targets
WHERE host_id = @host_id AND phase = @phase::santa_sync_target_phase;

-- name: DeleteSantaSyncPendingRules :exec
DELETE FROM santa_sync_pending_rules
WHERE host_id = @host_id;

-- name: ListSantaSyncTargets :many
SELECT
    rule_type::text,
    identifier,
    policy::text,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash
FROM santa_sync_targets
WHERE host_id = @host_id AND phase = @phase::santa_sync_target_phase
ORDER BY position;

-- name: InsertSantaSyncTarget :exec
INSERT INTO santa_sync_targets (
    host_id,
    phase,
    position,
    rule_type,
    identifier,
    policy,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash
)
VALUES (
    @host_id,
    @phase::santa_sync_target_phase,
    @position,
    @rule_type::santa_rule_type,
    @identifier,
    @policy::santa_policy,
    @cel_expression,
    @custom_message,
    @custom_url,
    @notification_app_name,
    @payload_hash
);

-- name: InsertSantaSyncPendingRule :exec
INSERT INTO santa_sync_pending_rules (
    host_id,
    position,
    rule_type,
    identifier,
    policy,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash,
    removed
)
VALUES (
    @host_id,
    @position,
    @rule_type::santa_rule_type,
    @identifier,
    sqlc.narg(policy)::santa_policy,
    @cel_expression,
    @custom_message,
    @custom_url,
    @notification_app_name,
    @payload_hash,
    @removed
);

-- name: ListSantaPendingPayloadPage :many
SELECT
    rule_type::text,
    identifier,
    COALESCE(policy::text, '')::text AS policy,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash,
    removed
FROM santa_sync_pending_rules
WHERE host_id = @host_id
ORDER BY position
LIMIT @limit_count OFFSET @offset_count;

-- name: GetSantaPendingState :one
SELECT pending_payload_rule_count, pending_full_sync
FROM santa_sync_state
WHERE host_id = @host_id;

-- name: MarkSantaSyncAttempt :exec
UPDATE santa_sync_state
SET
    client_rules_hash = @client_rules_hash,
    rules_received = @rules_received,
    rules_processed = @rules_processed,
    last_rule_sync_attempt_at = now(),
    updated_at = now()
WHERE host_id = @host_id;

-- name: PromoteSantaDesiredSyncTargets :exec
INSERT INTO santa_sync_targets (
    host_id,
    phase,
    position,
    rule_type,
    identifier,
    policy,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash,
    updated_at
)
SELECT
    host_id,
    'applied'::santa_sync_target_phase,
    position,
    rule_type,
    identifier,
    policy,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash,
    now()
FROM santa_sync_targets
WHERE santa_sync_targets.host_id = @promote_host_id AND santa_sync_targets.phase = 'desired'
ORDER BY position;

-- name: CompleteSantaSync :exec
UPDATE santa_sync_state
SET
    client_rules_hash = @client_rules_hash,
    rules_received = @rules_received,
    rules_processed = @rules_processed,
    pending_full_sync = false,
    pending_payload_rule_count = 0,
    pending_preflight_at = NULL,
    last_rule_sync_attempt_at = now(),
    last_rule_sync_success_at = now(),
    last_clean_sync_at = CASE WHEN @pending_full_sync::boolean THEN now() ELSE last_clean_sync_at END,
    updated_at = now()
WHERE host_id = @host_id;
