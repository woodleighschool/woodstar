-- +goose Up

CREATE INDEX santa_execution_events_occurred_at_idx
    ON santa_execution_events (occurred_at DESC, id DESC);

CREATE INDEX santa_file_access_events_occurred_at_idx
    ON santa_file_access_events (occurred_at DESC, id DESC);

CREATE INDEX santa_standalone_rule_creation_events_occurred_at_idx
    ON santa_standalone_rule_creation_events (occurred_at);
