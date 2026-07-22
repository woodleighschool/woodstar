-- +goose Up

CREATE TYPE target_direction AS ENUM ('include', 'exclude');

CREATE TABLE labels (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    builtin_key TEXT,
    description TEXT NOT NULL DEFAULT '',
    query TEXT,
    criteria JSONB,
    label_type TEXT NOT NULL CHECK (label_type IN ('builtin', 'regular')),
    label_membership_type TEXT NOT NULL CHECK (label_membership_type IN ('dynamic', 'manual', 'derived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (label_membership_type = 'dynamic' AND NULLIF(btrim(query), '') IS NOT NULL AND criteria IS NULL)
        OR (label_membership_type = 'manual' AND query IS NULL AND criteria IS NULL)
        OR (label_membership_type = 'derived' AND query IS NULL AND criteria IS NOT NULL)
    ),
    CHECK (
        (label_type = 'builtin' AND builtin_key IS NOT NULL AND builtin_key IN ('all-hosts'))
        OR (label_type = 'regular' AND builtin_key IS NULL)
    )
);

CREATE INDEX labels_label_type_idx ON labels (label_type);
CREATE INDEX labels_label_membership_type_idx ON labels (label_membership_type);
CREATE UNIQUE INDEX labels_builtin_key_unique_idx ON labels (builtin_key) WHERE builtin_key IS NOT NULL;

CREATE TABLE label_membership (
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (label_id, host_id)
);

CREATE INDEX label_membership_host_idx ON label_membership (host_id);

INSERT INTO labels (name, builtin_key, description, query, label_type, label_membership_type)
VALUES ('All Hosts', 'all-hosts', 'Every enrolled host.', NULL, 'builtin', 'manual');
