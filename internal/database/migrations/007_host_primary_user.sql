-- +goose Up

CREATE TYPE host_primary_user_source AS ENUM ('manual', 'orbit_profile');

CREATE INDEX users_lower_email_idx ON users (lower(email));
CREATE INDEX users_lower_upn_idx ON users (lower(user_principal_name))
    WHERE user_principal_name IS NOT NULL;

CREATE TABLE host_primary_user_sources (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    source host_primary_user_source NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, source)
);

INSERT INTO host_primary_user_sources (
    host_id,
    email,
    source,
    created_at,
    updated_at
)
SELECT
    host_id,
    email,
    source::text::host_primary_user_source,
    created_at,
    updated_at
FROM host_user_affinity_mappings
WHERE source IN ('manual', 'orbit_profile');

CREATE INDEX host_primary_user_sources_host_idx ON host_primary_user_sources (host_id);
CREATE INDEX host_primary_user_sources_email_idx ON host_primary_user_sources (email);

DROP TABLE host_user_links;
DROP TABLE host_user_affinity_mappings;
DROP TYPE host_user_link_source;
DROP TYPE host_user_affinity_source;
