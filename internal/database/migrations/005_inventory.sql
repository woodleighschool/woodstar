-- +goose Up

CREATE TABLE host_users (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    uid TEXT NOT NULL,
    username TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    directory TEXT NOT NULL DEFAULT '',
    shell TEXT NOT NULL DEFAULT '',
    UNIQUE (host_id, uid, username)
);

CREATE INDEX host_users_host_idx ON host_users (host_id);

CREATE TABLE host_batteries (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    serial_number TEXT NOT NULL,
    manufacturer TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    chemistry TEXT NOT NULL DEFAULT '',
    cycle_count INTEGER,
    health TEXT NOT NULL DEFAULT '',
    designed_capacity INTEGER,
    max_capacity INTEGER,
    current_capacity INTEGER,
    percent_remaining DOUBLE PRECISION,
    UNIQUE (host_id, serial_number)
);

CREATE INDEX host_batteries_host_idx ON host_batteries (host_id);

CREATE TABLE host_certificates (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    sha1 TEXT NOT NULL,
    common_name TEXT NOT NULL DEFAULT '',
    subject_country TEXT NOT NULL DEFAULT '',
    subject_organization TEXT NOT NULL DEFAULT '',
    subject_organizational_unit TEXT NOT NULL DEFAULT '',
    subject_common_name TEXT NOT NULL DEFAULT '',
    issuer_country TEXT NOT NULL DEFAULT '',
    issuer_organization TEXT NOT NULL DEFAULT '',
    issuer_organizational_unit TEXT NOT NULL DEFAULT '',
    issuer_common_name TEXT NOT NULL DEFAULT '',
    key_algorithm TEXT NOT NULL DEFAULT '',
    key_strength INTEGER,
    key_usage TEXT NOT NULL DEFAULT '',
    signing_algorithm TEXT NOT NULL DEFAULT '',
    not_valid_after TIMESTAMPTZ,
    not_valid_before TIMESTAMPTZ,
    serial TEXT NOT NULL DEFAULT '',
    certificate_authority BOOLEAN NOT NULL DEFAULT FALSE,
    source TEXT NOT NULL DEFAULT '',
    username TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    UNIQUE (host_id, sha1, source, username)
);

CREATE INDEX host_certificates_host_idx ON host_certificates (host_id);
CREATE INDEX host_certificates_sha1_idx ON host_certificates (sha1);

CREATE TABLE software_titles (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    extension_for TEXT NOT NULL DEFAULT '',
    bundle_identifier TEXT NOT NULL DEFAULT '',
    vendor TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (name, source, extension_for, bundle_identifier)
);

CREATE UNIQUE INDEX software_titles_bundle_idx
    ON software_titles (bundle_identifier, source, extension_for)
    WHERE bundle_identifier <> '';

CREATE TABLE software (
    id BIGSERIAL PRIMARY KEY,
    title_id BIGINT NOT NULL REFERENCES software_titles (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    bundle_identifier TEXT NOT NULL DEFAULT '',
    extension_id TEXT NOT NULL DEFAULT '',
    extension_for TEXT NOT NULL DEFAULT '',
    vendor TEXT NOT NULL DEFAULT '',
    arch TEXT NOT NULL DEFAULT '',
    release TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (
        title_id,
        version,
        source,
        bundle_identifier,
        extension_id,
        extension_for,
        vendor,
        arch,
        release
    )
);

CREATE INDEX software_name_idx ON software (name);
CREATE INDEX software_title_idx ON software (title_id);

CREATE TABLE host_software (
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    software_id BIGINT NOT NULL REFERENCES software (id) ON DELETE CASCADE,
    last_opened_at TIMESTAMPTZ,
    PRIMARY KEY (host_id, software_id)
);

CREATE INDEX host_software_software_idx ON host_software (software_id);

CREATE TABLE host_software_installed_paths (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    software_id BIGINT NOT NULL REFERENCES software (id) ON DELETE CASCADE,
    installed_path TEXT NOT NULL,
    team_identifier TEXT NOT NULL DEFAULT '',
    cdhash_sha256 TEXT,
    executable_sha256 TEXT,
    executable_path TEXT,
    UNIQUE (host_id, software_id, installed_path)
);

CREATE INDEX host_software_installed_paths_host_software_idx
    ON host_software_installed_paths (host_id, software_id);
