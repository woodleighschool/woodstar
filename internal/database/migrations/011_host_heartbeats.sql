-- +goose Up

CREATE TABLE host_heartbeats (
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('orbit', 'osquery', 'munki', 'santa')),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    remote_ip INET,
    user_agent TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (host_id, source)
);

CREATE INDEX host_heartbeats_source_seen_idx
    ON host_heartbeats (source, last_seen_at DESC);
