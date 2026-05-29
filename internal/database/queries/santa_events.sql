-- name: SweepSantaEventsBefore :one
WITH deleted_execution AS (
    DELETE FROM santa_execution_events
    WHERE santa_execution_events.occurred_at < @cutoff_time
    RETURNING 1
),
deleted_file_access AS (
    DELETE FROM santa_file_access_events
    WHERE santa_file_access_events.occurred_at < @cutoff_time
    RETURNING 1
)
SELECT
    (SELECT count(*) FROM deleted_execution)::integer
    + (SELECT count(*) FROM deleted_file_access)::integer AS deleted_count;

-- name: UpsertSantaExecutable :one
INSERT INTO santa_executables (
    sha256,
    file_name,
    file_bundle_id,
    file_bundle_path,
    file_bundle_executable_rel_path,
    file_bundle_name,
    file_bundle_version,
    file_bundle_version_string,
    file_bundle_hash,
    file_bundle_hash_millis,
    file_bundle_binary_count,
    signing_id,
    team_id,
    cdhash,
    codesigning_flags,
    signing_status,
    secure_signing_time,
    signing_time,
    entitlements,
    updated_at
)
VALUES (
    @sha256,
    @file_name,
    @file_bundle_id,
    @file_bundle_path,
    @file_bundle_executable_rel_path,
    @file_bundle_name,
    @file_bundle_version,
    @file_bundle_version_string,
    @file_bundle_hash,
    @file_bundle_hash_millis,
    @file_bundle_binary_count,
    @signing_id,
    @team_id,
    @cdhash,
    @codesigning_flags,
    @signing_status::santa_signing_status,
    sqlc.narg(secure_signing_time)::timestamptz,
    sqlc.narg(signing_time)::timestamptz,
    @entitlements,
    now()
)
ON CONFLICT (sha256) DO UPDATE SET
    file_name = EXCLUDED.file_name,
    file_bundle_id = EXCLUDED.file_bundle_id,
    file_bundle_path = EXCLUDED.file_bundle_path,
    file_bundle_executable_rel_path = EXCLUDED.file_bundle_executable_rel_path,
    file_bundle_name = EXCLUDED.file_bundle_name,
    file_bundle_version = EXCLUDED.file_bundle_version,
    file_bundle_version_string = EXCLUDED.file_bundle_version_string,
    file_bundle_hash = EXCLUDED.file_bundle_hash,
    file_bundle_hash_millis = EXCLUDED.file_bundle_hash_millis,
    file_bundle_binary_count = EXCLUDED.file_bundle_binary_count,
    signing_id = EXCLUDED.signing_id,
    team_id = EXCLUDED.team_id,
    cdhash = EXCLUDED.cdhash,
    codesigning_flags = EXCLUDED.codesigning_flags,
    signing_status = EXCLUDED.signing_status,
    secure_signing_time = EXCLUDED.secure_signing_time,
    signing_time = EXCLUDED.signing_time,
    entitlements = EXCLUDED.entitlements,
    updated_at = now()
RETURNING id;

-- name: UpsertSantaSigningChain :one
INSERT INTO santa_signing_chains (sha256)
VALUES (@sha256)
ON CONFLICT (sha256) DO UPDATE SET sha256 = EXCLUDED.sha256
RETURNING id;

-- name: UpsertSantaCertificate :one
INSERT INTO santa_certificates (
    sha256,
    common_name,
    organization,
    organizational_unit,
    valid_from,
    valid_until,
    updated_at
)
VALUES (
    @sha256,
    @common_name,
    @organization,
    @organizational_unit,
    sqlc.narg(valid_from)::timestamptz,
    sqlc.narg(valid_until)::timestamptz,
    now()
)
ON CONFLICT (sha256) DO UPDATE SET
    common_name = EXCLUDED.common_name,
    organization = EXCLUDED.organization,
    organizational_unit = EXCLUDED.organizational_unit,
    valid_from = EXCLUDED.valid_from,
    valid_until = EXCLUDED.valid_until,
    updated_at = now()
RETURNING id;

-- name: UpsertSantaSigningChainEntry :exec
INSERT INTO santa_signing_chain_entries (signing_chain_id, position, certificate_id)
VALUES (@signing_chain_id, @position, @certificate_id)
ON CONFLICT (signing_chain_id, position) DO UPDATE SET certificate_id = EXCLUDED.certificate_id;

-- name: LinkSantaExecutableSigningChain :exec
INSERT INTO santa_executable_signing_chains (executable_id, signing_chain_id)
VALUES (@executable_id, @signing_chain_id)
ON CONFLICT DO NOTHING;

-- name: UpsertSantaBundle :one
INSERT INTO santa_bundles (
    sha256,
    bundle_id,
    name,
    path,
    executable_rel_path,
    version,
    version_string,
    binary_count,
    hash_millis,
    updated_at
)
VALUES (
    @sha256,
    @bundle_id,
    @name,
    @path,
    @executable_rel_path,
    @version,
    @version_string,
    @binary_count,
    @hash_millis,
    now()
)
ON CONFLICT (sha256) DO UPDATE SET
    bundle_id = COALESCE(NULLIF(EXCLUDED.bundle_id, ''), santa_bundles.bundle_id),
    name = COALESCE(NULLIF(EXCLUDED.name, ''), santa_bundles.name),
    path = COALESCE(NULLIF(EXCLUDED.path, ''), santa_bundles.path),
    executable_rel_path = COALESCE(NULLIF(EXCLUDED.executable_rel_path, ''), santa_bundles.executable_rel_path),
    version = COALESCE(NULLIF(EXCLUDED.version, ''), santa_bundles.version),
    version_string = COALESCE(NULLIF(EXCLUDED.version_string, ''), santa_bundles.version_string),
    binary_count = CASE
        WHEN EXCLUDED.binary_count > 0 THEN EXCLUDED.binary_count
        ELSE santa_bundles.binary_count
    END,
    hash_millis = CASE
        WHEN EXCLUDED.hash_millis > 0 THEN EXCLUDED.hash_millis
        ELSE santa_bundles.hash_millis
    END,
    updated_at = now()
RETURNING id;

-- name: LinkSantaBundleExecutable :exec
INSERT INTO santa_bundle_executables (bundle_id, executable_id)
VALUES (@bundle_id, @executable_id)
ON CONFLICT DO NOTHING;

-- name: RefreshSantaBundleUploadedAt :exec
UPDATE santa_bundles b
SET uploaded_at = COALESCE(uploaded_at, now()), updated_at = now()
WHERE b.id = @bundle_id
  AND b.binary_count > 0
  AND (
      SELECT count(*)
      FROM santa_bundle_executables be
      WHERE be.bundle_id = b.id
  ) >= b.binary_count;

-- name: ListIncompleteSantaBundleHashes :many
SELECT b.sha256
FROM santa_bundles b
WHERE b.sha256 = ANY(@hashes::text[])
  AND b.uploaded_at IS NULL
ORDER BY b.sha256;

-- name: InsertSantaExecutionEvent :exec
INSERT INTO santa_execution_events (
    host_id,
    executable_id,
    file_path,
    executing_user,
    pid,
    ppid,
    parent_name,
    logged_in_users,
    current_sessions,
    decision,
    occurred_at
)
VALUES (
    @host_id,
    @executable_id,
    @file_path,
    @executing_user,
    @pid,
    @ppid,
    @parent_name,
    @logged_in_users,
    @current_sessions,
    @decision::santa_execution_decision,
    @occurred_at
);

-- name: InsertSantaFileAccessEvent :exec
INSERT INTO santa_file_access_events (
    host_id,
    rule_version,
    rule_name,
    target,
    decision,
    primary_process_sha256,
    primary_process_path,
    primary_process_signing_id,
    primary_process_team_id,
    primary_process_cdhash,
    primary_process_pid,
    process_chain,
    occurred_at
)
VALUES (
    @host_id,
    @rule_version,
    @rule_name,
    @target,
    @decision::santa_file_access_decision,
    @primary_process_sha256,
    @primary_process_path,
    @primary_process_signing_id,
    @primary_process_team_id,
    @primary_process_cdhash,
    @primary_process_pid,
    @process_chain,
    @occurred_at
);

-- name: GetSantaExecutionEvent :one
SELECT
    ee.id,
    h.id AS host_id,
    h.display_name,
    h.hostname,
    h.computer_name,
    h.hardware_serial,
    h.hardware_model,
    COALESCE(sh.machine_id, '') AS santa_machine_id,
    COALESCE(sh.santa_version, '') AS santa_version,
    COALESCE(sh.client_mode_reported::text, '')::text AS santa_client_mode,
    ee.file_path,
    ee.executing_user,
    ee.pid,
    ee.ppid,
    ee.parent_name,
    ee.logged_in_users,
    ee.current_sessions,
    ee.decision::text AS decision,
    ee.occurred_at,
    ee.ingested_at,
    e.id AS executable_id,
    e.sha256,
    e.file_name,
    e.file_bundle_id,
    e.file_bundle_path,
    e.file_bundle_executable_rel_path,
    e.file_bundle_name,
    e.file_bundle_version,
    e.file_bundle_version_string,
    e.file_bundle_hash,
    e.file_bundle_hash_millis,
    e.file_bundle_binary_count,
    e.signing_id,
    e.team_id,
    e.cdhash,
    e.codesigning_flags,
    e.signing_status::text AS signing_status,
    e.secure_signing_time,
    e.signing_time,
    e.entitlements,
    COALESCE((
        SELECT jsonb_agg(
            jsonb_build_object(
                'sha256', c.sha256,
                'common_name', c.common_name,
                'org', c.organization,
                'ou', c.organizational_unit,
                'valid_from', COALESCE(extract(epoch from c.valid_from)::integer, 0),
                'valid_until', COALESCE(extract(epoch from c.valid_until)::integer, 0)
            )
            ORDER BY sce.position
        )
        FROM (
            SELECT sc.id
            FROM santa_executable_signing_chains esc
            JOIN santa_signing_chains sc ON sc.id = esc.signing_chain_id
            WHERE esc.executable_id = e.id
            ORDER BY sc.first_seen_at DESC, sc.id DESC
            LIMIT 1
        ) latest_chain
        JOIN santa_signing_chain_entries sce ON sce.signing_chain_id = latest_chain.id
        JOIN santa_certificates c ON c.id = sce.certificate_id
    ), '[]'::jsonb)::text AS signing_chain
FROM santa_execution_events ee
JOIN santa_executables e ON e.id = ee.executable_id
JOIN hosts h ON h.id = ee.host_id
LEFT JOIN santa_hosts sh ON sh.host_id = h.id
WHERE ee.id = @id;

-- name: GetSantaFileAccessEvent :one
SELECT
    fae.id,
    h.id AS host_id,
    h.display_name,
    h.hostname,
    h.computer_name,
    h.hardware_serial,
    h.hardware_model,
    COALESCE(sh.machine_id, '') AS santa_machine_id,
    COALESCE(sh.santa_version, '') AS santa_version,
    COALESCE(sh.client_mode_reported::text, '')::text AS santa_client_mode,
    fae.rule_version,
    fae.rule_name,
    fae.target,
    fae.decision::text AS decision,
    fae.process_chain,
    fae.occurred_at,
    fae.ingested_at
FROM santa_file_access_events fae
JOIN hosts h ON h.id = fae.host_id
LEFT JOIN santa_hosts sh ON sh.host_id = h.id
WHERE fae.id = @id;
