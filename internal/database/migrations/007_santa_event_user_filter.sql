-- +goose Up

CREATE INDEX IF NOT EXISTS santa_execution_events_user_time_idx
    ON santa_execution_events (executing_user, occurred_at DESC);

-- +goose Down

DROP INDEX IF EXISTS santa_execution_events_user_time_idx;
