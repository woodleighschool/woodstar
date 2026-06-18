-- +goose Up

CREATE TABLE munki_distribution_points (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    position INT NOT NULL UNIQUE,
    client_cidrs TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    client_base_url TEXT NOT NULL DEFAULT '',
    key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE munki_distribution_package_states (
    distribution_point_id BIGINT NOT NULL
        REFERENCES munki_distribution_points (id) ON DELETE CASCADE,
    package_id BIGINT NOT NULL
        REFERENCES munki_packages (id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'syncing', 'current', 'error')),
    reported_sha256 TEXT,
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (distribution_point_id, package_id)
);

CREATE INDEX munki_distribution_package_states_package_idx
    ON munki_distribution_package_states (package_id);
