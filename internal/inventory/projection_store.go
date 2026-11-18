package inventory

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
)

// ReplaceHostSoftware reconciles the reported sources of a host's software inventory.
func (s *Store) ReplaceHostSoftware(ctx context.Context, hostID int64, snapshot SoftwareSnapshot) error {
	for _, entry := range snapshot.Entries {
		if entry.Name != "" && (!entry.Source.Valid() || !slices.Contains(snapshot.Sources, entry.Source)) {
			return fmt.Errorf("unsupported software source %q", entry.Source)
		}
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := stageSoftwareSnapshot(ctx, tx, snapshot.Entries); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM hosts WHERE id = $1 FOR NO KEY UPDATE`, hostID).Scan(&hostID); err != nil {
			return err
		}
		if err := reconcileSoftwareCatalog(ctx, tx); err != nil {
			return err
		}
		return reconcileHostSoftware(ctx, tx, hostID, snapshot)
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
    signature_signed boolean,
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
		if entry.Name != "" {
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
			"signature_signed",
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
			var signatureSigned *bool
			var signature SoftwareCodeSignature
			if entry.Signature != nil {
				signature = *entry.Signature
				signatureSigned = &signature.Signed
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
				signatureSigned,
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
	// A bundle ID groups title names that differ across installations.
	const titleMatch = `title.source = observed.source AND title.extension_for = observed.extension_for
    AND title.bundle_identifier = observed.bundle_identifier
    AND (observed.bundle_identifier <> '' OR title.name = observed.name)`

	if _, err := tx.Exec(ctx, `
WITH observed AS (
    SELECT DISTINCT ON (source, extension_for, bundle_identifier, CASE WHEN bundle_identifier = '' THEN name END)
        name, source, extension_for, bundle_identifier, vendor
    FROM inventory_software_snapshot
    ORDER BY source, extension_for, bundle_identifier, CASE WHEN bundle_identifier = '' THEN name END, position
)
INSERT INTO software_titles (name, source, extension_for, bundle_identifier, vendor)
SELECT observed.name, observed.source, observed.extension_for, observed.bundle_identifier, observed.vendor
FROM observed
WHERE NOT EXISTS (SELECT FROM software_titles title WHERE `+titleMatch+`)
ORDER BY observed.source, observed.extension_for, observed.bundle_identifier, observed.name
ON CONFLICT DO NOTHING`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
UPDATE inventory_software_snapshot observed SET title_id = title.id
FROM software_titles title
WHERE `+titleMatch); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
WITH vendors AS (
    SELECT DISTINCT ON (title_id) title_id, vendor FROM inventory_software_snapshot
    WHERE vendor <> '' ORDER BY title_id, position DESC
), changed AS MATERIALIZED (
    SELECT title.id, vendors.vendor FROM vendors
    JOIN software_titles title ON title.id = vendors.title_id
    WHERE title.vendor <> vendors.vendor
    ORDER BY title.id FOR NO KEY UPDATE OF title
)
UPDATE software_titles title SET vendor = changed.vendor, updated_at = now()
FROM changed WHERE title.id = changed.id`); err != nil {
		return err
	}

	const versionKey = `title_id, version, source, bundle_identifier, extension_id, extension_for, vendor, arch, release`
	if _, err := tx.Exec(ctx, `
WITH observed AS (
    SELECT DISTINCT ON (`+versionKey+`) name, `+versionKey+`
    FROM inventory_software_snapshot
    ORDER BY `+versionKey+`, position
)
INSERT INTO software (name, `+versionKey+`)
SELECT observed.name, `+versionKey+`
FROM observed LEFT JOIN software USING (`+versionKey+`)
WHERE software.id IS NULL
ORDER BY `+versionKey+`
ON CONFLICT DO NOTHING`); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
UPDATE inventory_software_snapshot observed SET software_id = software.id
FROM software
WHERE (software.title_id, software.version, software.source, software.bundle_identifier,
       software.extension_id, software.extension_for, software.vendor, software.arch, software.release) =
      (observed.title_id, observed.version, observed.source, observed.bundle_identifier,
       observed.extension_id, observed.extension_for, observed.vendor, observed.arch, observed.release)`)
	return err
}

func reconcileHostSoftware(ctx context.Context, tx pgx.Tx, hostID int64, snapshot SoftwareSnapshot) error {
	if _, err := tx.Exec(ctx, `
DELETE FROM host_software membership USING software
WHERE membership.host_id = $1 AND membership.software_id = software.id AND software.source = ANY($2)
  AND NOT EXISTS (SELECT FROM inventory_software_snapshot WHERE software_id = software.id)`, hostID, snapshot.Sources); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
MERGE INTO host_software current
USING (
    SELECT DISTINCT ON (software_id) software_id, last_opened_at FROM inventory_software_snapshot
    ORDER BY software_id, position DESC
) observed ON current.host_id = $1 AND current.software_id = observed.software_id
WHEN MATCHED AND current.last_opened_at IS DISTINCT FROM observed.last_opened_at
    THEN UPDATE SET last_opened_at = observed.last_opened_at
WHEN NOT MATCHED THEN INSERT (host_id, software_id, last_opened_at)
    VALUES ($1, observed.software_id, observed.last_opened_at)`, hostID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM host_software_installed_paths path
WHERE path.host_id = $1
  AND EXISTS (SELECT FROM inventory_software_snapshot observed WHERE observed.software_id = path.software_id)
  AND NOT EXISTS (
      SELECT FROM inventory_software_snapshot observed
      WHERE observed.software_id = path.software_id AND observed.installed_path = path.installed_path
  )`, hostID); err != nil {
		return err
	}
	return reconcileHostSoftwarePaths(ctx, tx, hostID, snapshot)
}

func reconcileHostSoftwarePaths(ctx context.Context, tx pgx.Tx, hostID int64, snapshot SoftwareSnapshot) error {
	const paths = `WITH observed AS (
    SELECT DISTINCT ON (software_id, installed_path)
        software_id, installed_path, signature_signed, identifier, signing_authority, team_identifier,
        NULLIF(cdhash, '') AS cdhash, NULLIF(executable_sha256, '') AS executable_sha256, NULLIF(executable_path, '') AS executable_path
    FROM inventory_software_snapshot WHERE installed_path <> ''
    ORDER BY software_id, installed_path, position DESC
)
`
	const pathMatch = `path.host_id = $1 AND path.software_id = observed.software_id AND path.installed_path = observed.installed_path`
	if _, err := tx.Exec(ctx, paths+`
INSERT INTO host_software_installed_paths (
    host_id, software_id, installed_path, signature_signed, identifier, signing_authority, team_identifier,
    cdhash, executable_sha256, executable_path
)
SELECT $1, observed.* FROM observed
WHERE NOT EXISTS (SELECT FROM host_software_installed_paths path WHERE `+pathMatch+`)`, hostID); err != nil {
		return err
	}
	if snapshot.SignaturesReported {
		const signature = `(observed.signature_signed, observed.identifier, observed.signing_authority, observed.team_identifier, observed.cdhash)`
		if _, err := tx.Exec(ctx, paths+`
UPDATE host_software_installed_paths path SET (signature_signed, identifier, signing_authority, team_identifier, cdhash) = `+signature+`
FROM observed WHERE `+pathMatch+`
  AND (path.signature_signed, path.identifier, path.signing_authority, path.team_identifier, path.cdhash)
      IS DISTINCT FROM `+signature, hostID); err != nil {
			return err
		}
	}
	if snapshot.ExecutableHashesReported {
		_, err := tx.Exec(ctx, paths+`
UPDATE host_software_installed_paths path
SET (executable_sha256, executable_path) = (observed.executable_sha256, observed.executable_path)
FROM observed WHERE `+pathMatch+`
  AND (path.executable_sha256, path.executable_path) IS DISTINCT FROM (observed.executable_sha256, observed.executable_path)`, hostID)
		return err
	}
	return nil
}
