-- +goose Up

ALTER TABLE santa_execution_events
    ADD COLUMN static_rule BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE santa_standalone_rule_creation_events (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    identifier TEXT NOT NULL,
    decision santa_execution_decision NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(identifier), '') IS NOT NULL)
);

CREATE INDEX santa_standalone_rule_creation_events_host_time_idx
    ON santa_standalone_rule_creation_events (host_id, occurred_at DESC);

ALTER TABLE santa_sync_state
    ADD COLUMN preflight_rules_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN confirmed_rules_hash TEXT NOT NULL DEFAULT '',
    ADD CONSTRAINT santa_sync_state_preflight_rules_hash_check
        CHECK (preflight_rules_hash = '' OR preflight_rules_hash ~ '^[0-9a-f]{32}$'),
    ADD CONSTRAINT santa_sync_state_confirmed_rules_hash_check
        CHECK (confirmed_rules_hash = '' OR confirmed_rules_hash ~ '^[0-9a-f]{32}$');

ALTER TABLE santa_sync_targets
    DROP CONSTRAINT santa_sync_targets_host_id_phase_payload_hash_key,
    ADD CONSTRAINT santa_sync_targets_host_phase_identity_key
        UNIQUE (host_id, phase, rule_type, identifier);
