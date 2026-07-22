-- +goose Up

CREATE TABLE osquery_reports (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    min_osquery_version TEXT,
    schedule_interval INTEGER NOT NULL DEFAULT 0,
    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (schedule_interval >= 0)
);

CREATE INDEX osquery_reports_schedule_idx
    ON osquery_reports (schedule_interval)
    WHERE schedule_interval > 0;

CREATE TABLE osquery_report_results (
    id BIGSERIAL PRIMARY KEY,
    report_id BIGINT NOT NULL REFERENCES osquery_reports (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    data JSONB,
    last_fetched TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX osquery_report_results_report_last_fetched_idx
    ON osquery_report_results (report_id, last_fetched);
CREATE INDEX osquery_report_results_report_host_last_fetched_idx
    ON osquery_report_results (report_id, host_id, last_fetched);

CREATE TABLE osquery_checks (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE osquery_check_membership (
    check_id BIGINT NOT NULL REFERENCES osquery_checks (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    passes BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (check_id, host_id)
);

CREATE INDEX osquery_check_membership_passes_idx
    ON osquery_check_membership (check_id, passes);

CREATE TABLE osquery_report_targets (
    report_id BIGINT NOT NULL REFERENCES osquery_reports (id) ON DELETE CASCADE,
    direction target_direction NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE RESTRICT,
    PRIMARY KEY (report_id, direction, position),
    UNIQUE (report_id, label_id)
);

CREATE INDEX osquery_report_targets_label_idx ON osquery_report_targets (label_id);

CREATE TABLE osquery_check_targets (
    check_id BIGINT NOT NULL REFERENCES osquery_checks (id) ON DELETE CASCADE,
    direction target_direction NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE RESTRICT,
    PRIMARY KEY (check_id, direction, position),
    UNIQUE (check_id, label_id)
);

CREATE INDEX osquery_check_targets_label_idx ON osquery_check_targets (label_id);
