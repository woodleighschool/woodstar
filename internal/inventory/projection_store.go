package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ReplaceHostSoftware replaces a host's software snapshot in one transaction.
func (s *Store) ReplaceHostSoftware(ctx context.Context, hostID int64, entries []HostSoftwareEntry) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := stageSoftwareSnapshot(ctx, tx, entries); err != nil {
			return err
		}
		if err := reconcileSoftwareCatalog(ctx, tx); err != nil {
			return err
		}
		return replaceHostSoftwareSnapshot(ctx, tx, hostID)
	})
}

func stageSoftwareSnapshot(ctx context.Context, tx pgx.Tx, entries []HostSoftwareEntry) error {
	if _, err := tx.Exec(ctx, `
CREATE TEMP TABLE IF NOT EXISTS inventory_software_snapshot (
    position bigint NOT NULL,
    name text NOT NULL,
    version text NOT NULL,
    source text NOT NULL,
    bundle_identifier text NOT NULL,
    extension_id text NOT NULL,
    extension_for text NOT NULL,
    vendor text NOT NULL,
    arch text NOT NULL,
    release text NOT NULL,
    installed_path text NOT NULL,
    signature_valid boolean,
    identifier text NOT NULL,
    signing_authority text NOT NULL,
    team_identifier text NOT NULL,
    cdhash text NOT NULL,
    executable_sha256 text NOT NULL,
    executable_path text NOT NULL,
    last_opened_at timestamptz,
    title_id bigint,
    software_id bigint
) ON COMMIT DELETE ROWS`); err != nil {
		return err
	}

	validEntries := make([]HostSoftwareEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Name != "" && entry.Source != "" {
			validEntries = append(validEntries, entry)
		}
	}
	if len(validEntries) == 0 {
		return nil
	}

	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"inventory_software_snapshot"},
		[]string{
			"position",
			"name",
			"version",
			"source",
			"bundle_identifier",
			"extension_id",
			"extension_for",
			"vendor",
			"arch",
			"release",
			"installed_path",
			"signature_valid",
			"identifier",
			"signing_authority",
			"team_identifier",
			"cdhash",
			"executable_sha256",
			"executable_path",
			"last_opened_at",
		},
		pgx.CopyFromSlice(len(validEntries), func(i int) ([]any, error) {
			entry := validEntries[i]
			var signatureValid *bool
			var signature SoftwareCodeSignature
			if entry.Signature != nil {
				signature = *entry.Signature
				signatureValid = &signature.Valid
			}
			return []any{
				int64(i),
				entry.Name,
				entry.Version,
				entry.Source,
				entry.BundleIdentifier,
				entry.ExtensionID,
				entry.ExtensionFor,
				entry.Vendor,
				entry.Arch,
				entry.Release,
				entry.InstalledPath,
				signatureValid,
				signature.Identifier,
				signature.Authority,
				signature.TeamIdentifier,
				signature.CDHash,
				entry.ExecutableSHA256,
				entry.ExecutablePath,
				entry.LastOpenedAt,
			}, nil
		}),
	)
	return err
}

func reconcileSoftwareCatalog(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `
INSERT INTO software_titles (name, source, extension_for, bundle_identifier, vendor)
SELECT
    (array_agg(name ORDER BY position))[1],
    source,
    extension_for,
    bundle_identifier,
    (array_agg(vendor ORDER BY (vendor <> '') DESC, position DESC))[1]
FROM inventory_software_snapshot
WHERE bundle_identifier <> ''
GROUP BY source, extension_for, bundle_identifier
ON CONFLICT (bundle_identifier, source, extension_for)
WHERE bundle_identifier <> ''
DO UPDATE SET
    vendor = COALESCE(NULLIF(EXCLUDED.vendor, ''), software_titles.vendor),
    updated_at = now()`); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO software_titles (name, source, extension_for, bundle_identifier, vendor)
SELECT
    name,
    source,
    extension_for,
    bundle_identifier,
    (array_agg(vendor ORDER BY (vendor <> '') DESC, position DESC))[1]
FROM inventory_software_snapshot
WHERE bundle_identifier = ''
GROUP BY name, source, extension_for, bundle_identifier
ON CONFLICT (name, source, extension_for, bundle_identifier) DO UPDATE SET
    vendor = COALESCE(NULLIF(EXCLUDED.vendor, ''), software_titles.vendor),
    updated_at = now()`); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
UPDATE inventory_software_snapshot snapshot
SET title_id = title.id
FROM software_titles title
WHERE title.source = snapshot.source
  AND title.extension_for = snapshot.extension_for
  AND (
      (
          snapshot.bundle_identifier <> ''
          AND title.bundle_identifier = snapshot.bundle_identifier
      )
      OR (
          snapshot.bundle_identifier = ''
          AND title.bundle_identifier = ''
          AND title.name = snapshot.name
      )
  )`); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO software (
    title_id, name, version, source,
    bundle_identifier, extension_id, extension_for,
    vendor, arch, release
)
SELECT DISTINCT ON (
    title_id, version, source,
    bundle_identifier, extension_id, extension_for,
    vendor, arch, release
)
    title_id, name, version, source,
    bundle_identifier, extension_id, extension_for,
    vendor, arch, release
FROM inventory_software_snapshot
ORDER BY
    title_id, version, source,
    bundle_identifier, extension_id, extension_for,
    vendor, arch, release,
    position
ON CONFLICT (
    title_id, version, source,
    bundle_identifier, extension_id, extension_for,
    vendor, arch, release
) DO UPDATE SET updated_at = now()`); err != nil {
		return err
	}

	_, err := tx.Exec(ctx, `
UPDATE inventory_software_snapshot snapshot
SET software_id = software.id
FROM software
WHERE software.title_id = snapshot.title_id
  AND software.version = snapshot.version
  AND software.source = snapshot.source
  AND software.bundle_identifier = snapshot.bundle_identifier
  AND software.extension_id = snapshot.extension_id
  AND software.extension_for = snapshot.extension_for
  AND software.vendor = snapshot.vendor
  AND software.arch = snapshot.arch
  AND software.release = snapshot.release`)
	return err
}

func replaceHostSoftwareSnapshot(ctx context.Context, tx pgx.Tx, hostID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM host_software_installed_paths WHERE host_id = $1`, hostID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM host_software WHERE host_id = $1`, hostID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO host_software (host_id, software_id, last_opened_at)
SELECT DISTINCT ON (software_id)
    $1,
    software_id,
    last_opened_at
FROM inventory_software_snapshot
ORDER BY software_id, position DESC
ON CONFLICT (host_id, software_id) DO UPDATE SET
    last_opened_at = EXCLUDED.last_opened_at`, hostID); err != nil {
		return err
	}

	_, err := tx.Exec(ctx, `
INSERT INTO host_software_installed_paths (
    host_id, software_id, installed_path,
    signature_valid, identifier, signing_authority, team_identifier, cdhash,
    executable_sha256, executable_path
)
SELECT DISTINCT ON (software_id, installed_path)
    $1,
    software_id,
    installed_path,
    signature_valid,
    identifier,
    signing_authority,
    team_identifier,
    NULLIF(cdhash, ''),
    NULLIF(executable_sha256, ''),
    NULLIF(executable_path, '')
FROM inventory_software_snapshot
WHERE installed_path <> ''
ORDER BY software_id, installed_path, position DESC
ON CONFLICT (host_id, software_id, installed_path) DO UPDATE SET
    signature_valid = EXCLUDED.signature_valid,
    identifier = EXCLUDED.identifier,
    signing_authority = EXCLUDED.signing_authority,
    team_identifier = EXCLUDED.team_identifier,
    cdhash = EXCLUDED.cdhash,
    executable_sha256 = EXCLUDED.executable_sha256,
    executable_path = EXCLUDED.executable_path`, hostID)
	return err
}
