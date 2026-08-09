-- +goose Up

CREATE TABLE munki_distribution_worker_sessions (
    distribution_point_id BIGINT PRIMARY KEY
        REFERENCES munki_distribution_points (id) ON DELETE CASCADE,
    connection_id TEXT NOT NULL,
    protocol_version INT NOT NULL,
    build_version TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE munki_distribution_worker_rejections (
    distribution_point_id BIGINT PRIMARY KEY
        REFERENCES munki_distribution_points (id) ON DELETE CASCADE,
    protocol_version INT,
    build_version TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
