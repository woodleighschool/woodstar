-- +goose Up

ALTER TABLE santa_sync_targets
    DROP COLUMN position,
    DROP CONSTRAINT santa_sync_targets_host_phase_identity_key,
    ADD PRIMARY KEY (host_id, phase, rule_type, identifier);

DROP INDEX santa_sync_targets_host_phase_idx;
