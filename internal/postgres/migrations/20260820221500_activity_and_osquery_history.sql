-- +goose Up

CREATE TABLE activity_events (
    id BIGSERIAL PRIMARY KEY,
    area TEXT NOT NULL,
    action TEXT NOT NULL,
    actor_kind TEXT NOT NULL CHECK (actor_kind IN ('user', 'system')),
    actor_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    actor_name TEXT NOT NULL DEFAULT '',
    actor_email TEXT NOT NULL DEFAULT '',
    subject_type TEXT NOT NULL,
    subject_id BIGINT,
    subject_name TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX activity_events_occurred_at_idx
    ON activity_events (occurred_at DESC, id DESC);
CREATE INDEX activity_events_area_occurred_at_idx
    ON activity_events (area, occurred_at DESC, id DESC);
CREATE INDEX activity_events_subject_occurred_at_idx
    ON activity_events (subject_type, subject_id, occurred_at DESC, id DESC)
    WHERE subject_id IS NOT NULL;

CREATE TABLE osquery_host_status_points (
    bucket TIMESTAMPTZ PRIMARY KEY,
    online_count INTEGER NOT NULL CHECK (online_count >= 0),
    offline_count INTEGER NOT NULL CHECK (offline_count >= 0)
);

CREATE TABLE osquery_policy_status_points (
    policy_id BIGINT NOT NULL REFERENCES osquery_policies (id) ON DELETE CASCADE,
    bucket TIMESTAMPTZ NOT NULL,
    pass_count INTEGER NOT NULL CHECK (pass_count >= 0),
    fail_count INTEGER NOT NULL CHECK (fail_count >= 0),
    error_count INTEGER NOT NULL CHECK (error_count >= 0),
    pending_count INTEGER NOT NULL CHECK (pending_count >= 0),
    PRIMARY KEY (policy_id, bucket)
);

CREATE INDEX osquery_policy_status_points_bucket_idx
    ON osquery_policy_status_points (bucket);
