-- +goose NO TRANSACTION
-- +goose Up

ALTER TYPE santa_rule_type ADD VALUE IF NOT EXISTS 'bundle';

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'santa_signing_status'
    ) THEN
        CREATE TYPE santa_signing_status AS ENUM (
            'unspecified',
            'unsigned',
            'invalid',
            'adhoc',
            'development',
            'production'
        );
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE santa_sync_targets
    ADD COLUMN IF NOT EXISTS notification_app_name TEXT NOT NULL DEFAULT '';

ALTER TABLE santa_sync_pending_rules
    ADD COLUMN IF NOT EXISTS notification_app_name TEXT NOT NULL DEFAULT '';

ALTER TABLE santa_executables
    ADD COLUMN IF NOT EXISTS file_bundle_executable_rel_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS file_bundle_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS file_bundle_version TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS file_bundle_version_string TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS file_bundle_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS file_bundle_hash_millis INT NOT NULL DEFAULT 0 CHECK (file_bundle_hash_millis >= 0),
    ADD COLUMN IF NOT EXISTS file_bundle_binary_count INT NOT NULL DEFAULT 0 CHECK (file_bundle_binary_count >= 0),
    ADD COLUMN IF NOT EXISTS codesigning_flags BIGINT NOT NULL DEFAULT 0 CHECK (codesigning_flags >= 0),
    ADD COLUMN IF NOT EXISTS signing_status santa_signing_status NOT NULL DEFAULT 'unspecified',
    ADD COLUMN IF NOT EXISTS secure_signing_time TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS signing_time TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS santa_certificates (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    common_name TEXT NOT NULL DEFAULT '',
    organization TEXT NOT NULL DEFAULT '',
    organizational_unit TEXT NOT NULL DEFAULT '',
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(sha256), '') IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS santa_signing_chain_entries (
    signing_chain_id BIGINT NOT NULL REFERENCES santa_signing_chains (id) ON DELETE CASCADE,
    position INT NOT NULL CHECK (position >= 0),
    certificate_id BIGINT NOT NULL REFERENCES santa_certificates (id) ON DELETE RESTRICT,
    PRIMARY KEY (signing_chain_id, position)
);

CREATE INDEX IF NOT EXISTS santa_signing_chain_entries_certificate_idx
    ON santa_signing_chain_entries (certificate_id);

INSERT INTO santa_certificates (
    sha256,
    common_name,
    organization,
    organizational_unit,
    valid_from,
    valid_until
)
SELECT DISTINCT ON (chain.entry->>'sha256')
    chain.entry->>'sha256',
    COALESCE(chain.entry->>'common_name', ''),
    COALESCE(chain.entry->>'org', ''),
    COALESCE(chain.entry->>'ou', ''),
    CASE
        WHEN COALESCE(chain.entry->>'valid_from', '') ~ '^[0-9]+$'
            AND (chain.entry->>'valid_from')::bigint > 0
        THEN to_timestamp((chain.entry->>'valid_from')::double precision)
    END,
    CASE
        WHEN COALESCE(chain.entry->>'valid_until', '') ~ '^[0-9]+$'
            AND (chain.entry->>'valid_until')::bigint > 0
        THEN to_timestamp((chain.entry->>'valid_until')::double precision)
    END
FROM santa_signing_chains sc
CROSS JOIN LATERAL jsonb_array_elements(sc.entries) WITH ORDINALITY AS chain(entry, ordinality)
WHERE NULLIF(btrim(chain.entry->>'sha256'), '') IS NOT NULL
ORDER BY chain.entry->>'sha256', sc.id, chain.ordinality
ON CONFLICT (sha256) DO UPDATE SET
    common_name = EXCLUDED.common_name,
    organization = EXCLUDED.organization,
    organizational_unit = EXCLUDED.organizational_unit,
    valid_from = EXCLUDED.valid_from,
    valid_until = EXCLUDED.valid_until,
    updated_at = now();

INSERT INTO santa_signing_chain_entries (signing_chain_id, position, certificate_id)
SELECT
    sc.id,
    (chain.ordinality - 1)::int,
    c.id
FROM santa_signing_chains sc
CROSS JOIN LATERAL jsonb_array_elements(sc.entries) WITH ORDINALITY AS chain(entry, ordinality)
JOIN santa_certificates c ON c.sha256 = chain.entry->>'sha256'
WHERE NULLIF(btrim(chain.entry->>'sha256'), '') IS NOT NULL
ON CONFLICT (signing_chain_id, position) DO UPDATE SET
    certificate_id = EXCLUDED.certificate_id;

ALTER TABLE santa_signing_chains
    DROP COLUMN IF EXISTS entries;

ALTER TABLE santa_execution_events
    ADD COLUMN IF NOT EXISTS pid INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ppid INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS parent_name TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS santa_bundles (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    bundle_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    executable_rel_path TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    version_string TEXT NOT NULL DEFAULT '',
    binary_count INT NOT NULL DEFAULT 0 CHECK (binary_count >= 0),
    hash_millis INT NOT NULL DEFAULT 0 CHECK (hash_millis >= 0),
    uploaded_at TIMESTAMPTZ,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(sha256), '') IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS santa_bundle_executables (
    bundle_id BIGINT NOT NULL REFERENCES santa_bundles (id) ON DELETE CASCADE,
    executable_id BIGINT NOT NULL REFERENCES santa_executables (id) ON DELETE CASCADE,
    PRIMARY KEY (bundle_id, executable_id)
);

CREATE INDEX IF NOT EXISTS santa_bundle_executables_executable_idx
    ON santa_bundle_executables (executable_id);

ALTER TABLE santa_file_access_events
    ADD COLUMN IF NOT EXISTS primary_process_sha256 TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS primary_process_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS primary_process_signing_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS primary_process_team_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS primary_process_cdhash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS primary_process_pid INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS santa_file_access_events_primary_process_sha_idx
    ON santa_file_access_events (primary_process_sha256)
    WHERE primary_process_sha256 <> '';

-- +goose Down

DROP INDEX IF EXISTS santa_file_access_events_primary_process_sha_idx;

ALTER TABLE santa_file_access_events
    DROP COLUMN IF EXISTS primary_process_pid,
    DROP COLUMN IF EXISTS primary_process_cdhash,
    DROP COLUMN IF EXISTS primary_process_team_id,
    DROP COLUMN IF EXISTS primary_process_signing_id,
    DROP COLUMN IF EXISTS primary_process_path,
    DROP COLUMN IF EXISTS primary_process_sha256;

DROP TABLE IF EXISTS santa_bundle_executables;
DROP TABLE IF EXISTS santa_bundles;

ALTER TABLE santa_execution_events
    DROP COLUMN IF EXISTS parent_name,
    DROP COLUMN IF EXISTS ppid,
    DROP COLUMN IF EXISTS pid;

ALTER TABLE santa_signing_chains
    ADD COLUMN IF NOT EXISTS entries JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(entries) = 'array');

UPDATE santa_signing_chains sc
SET entries = COALESCE(chain_entries.entries, '[]'::JSONB)
FROM (
    SELECT
        sce.signing_chain_id,
        jsonb_agg(
            jsonb_build_object(
                'sha256', c.sha256,
                'common_name', c.common_name,
                'org', c.organization,
                'ou', c.organizational_unit,
                'valid_from', COALESCE(floor(extract(epoch FROM c.valid_from))::bigint, 0),
                'valid_until', COALESCE(floor(extract(epoch FROM c.valid_until))::bigint, 0)
            )
            ORDER BY sce.position
        ) AS entries
    FROM santa_signing_chain_entries sce
    JOIN santa_certificates c ON c.id = sce.certificate_id
    GROUP BY sce.signing_chain_id
) chain_entries
WHERE chain_entries.signing_chain_id = sc.id;

ALTER TABLE santa_signing_chains
    ALTER COLUMN entries DROP DEFAULT;

DROP TABLE IF EXISTS santa_signing_chain_entries;
DROP TABLE IF EXISTS santa_certificates;

ALTER TABLE santa_executables
    DROP COLUMN IF EXISTS signing_time,
    DROP COLUMN IF EXISTS secure_signing_time,
    DROP COLUMN IF EXISTS signing_status,
    DROP COLUMN IF EXISTS codesigning_flags,
    DROP COLUMN IF EXISTS file_bundle_binary_count,
    DROP COLUMN IF EXISTS file_bundle_hash_millis,
    DROP COLUMN IF EXISTS file_bundle_hash,
    DROP COLUMN IF EXISTS file_bundle_version_string,
    DROP COLUMN IF EXISTS file_bundle_version,
    DROP COLUMN IF EXISTS file_bundle_name,
    DROP COLUMN IF EXISTS file_bundle_executable_rel_path;

DROP TYPE IF EXISTS santa_signing_status;

ALTER TABLE santa_sync_pending_rules
    DROP COLUMN IF EXISTS notification_app_name;

ALTER TABLE santa_sync_targets
    DROP COLUMN IF EXISTS notification_app_name;

DELETE FROM santa_sync_pending_rules
WHERE rule_type::text = 'bundle';

DELETE FROM santa_sync_targets
WHERE rule_type::text = 'bundle';

DELETE FROM santa_rules
WHERE rule_type::text = 'bundle';

ALTER TYPE santa_rule_type RENAME TO santa_rule_type_with_bundle;
CREATE TYPE santa_rule_type AS ENUM ('binary', 'certificate', 'teamid', 'signingid', 'cdhash');

ALTER TABLE santa_rules
    ALTER COLUMN rule_type TYPE santa_rule_type USING rule_type::text::santa_rule_type;

ALTER TABLE santa_sync_targets
    ALTER COLUMN rule_type TYPE santa_rule_type USING rule_type::text::santa_rule_type;

ALTER TABLE santa_sync_pending_rules
    ALTER COLUMN rule_type TYPE santa_rule_type USING rule_type::text::santa_rule_type;

DROP TYPE santa_rule_type_with_bundle;
