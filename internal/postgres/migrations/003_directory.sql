-- +goose Up

CREATE TABLE directory_groups (
    id BIGSERIAL PRIMARY KEY,
    source directory_source NOT NULL,
    external_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    mail_nickname TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CHECK (source <> 'local')
);

CREATE TABLE directory_group_memberships (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES directory_groups (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX directory_group_memberships_group_idx ON directory_group_memberships (group_id);

CREATE TABLE directory_user_links (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    source directory_source NOT NULL,
    external_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, source),
    UNIQUE (source, external_id),
    CHECK (source <> 'local')
);
