-- +goose Up
CREATE TABLE queries (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    platform TEXT,
    min_osquery_version TEXT,
    schedule_interval INTEGER NOT NULL DEFAULT 0,
    logging_type TEXT NOT NULL DEFAULT 'snapshot' CHECK (logging_type IN ('snapshot')),
    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (schedule_interval >= 0)
);

CREATE INDEX queries_schedule_idx
    ON queries (schedule_interval)
    WHERE schedule_interval > 0;

CREATE TABLE query_results (
    id BIGSERIAL PRIMARY KEY,
    query_id BIGINT NOT NULL REFERENCES queries (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    data JSONB,
    last_fetched TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX query_results_query_last_fetched_idx
    ON query_results (query_id, last_fetched);

CREATE INDEX query_results_query_host_last_fetched_idx
    ON query_results (query_id, host_id, last_fetched);

CREATE TABLE checks (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    resolution TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    platform TEXT,
    min_osquery_version TEXT,
    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE check_membership (
    check_id BIGINT NOT NULL REFERENCES checks (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    passes BOOLEAN,
    first_failed_at TIMESTAMPTZ,
    last_evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (check_id, host_id)
);

CREATE INDEX check_membership_passes_idx
    ON check_membership (check_id, passes);

CREATE TABLE query_labels (
    query_id BIGINT NOT NULL REFERENCES queries (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    exclude BOOLEAN NOT NULL DEFAULT FALSE,
    require_all BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (query_id, label_id)
);

CREATE INDEX query_labels_label_idx ON query_labels (label_id);

CREATE TABLE check_labels (
    check_id BIGINT NOT NULL REFERENCES checks (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    exclude BOOLEAN NOT NULL DEFAULT FALSE,
    require_all BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (check_id, label_id)
);

CREATE INDEX check_labels_label_idx ON check_labels (label_id);

-- +goose Down
DROP TABLE check_labels;
DROP TABLE query_labels;
DROP TABLE check_membership;
DROP TABLE checks;
DROP TABLE query_results;
DROP TABLE queries;
