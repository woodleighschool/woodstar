package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ReplaceHostSoftware replaces a host's software snapshot in one transaction.
func (s *Store) ReplaceHostSoftware(ctx context.Context, hostID int64, entries []HostSoftwareEntry) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := resetHostSoftware(ctx, tx, hostID); err != nil {
			return err
		}
		for _, entry := range entries {
			if err := replaceHostSoftwareEntry(ctx, tx, hostID, entry); err != nil {
				return err
			}
		}
		return nil
	})
}

func resetHostSoftware(ctx context.Context, tx pgx.Tx, hostID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM host_software_installed_paths WHERE host_id = $1`, hostID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM host_software WHERE host_id = $1`, hostID)
	return err
}

func replaceHostSoftwareEntry(ctx context.Context, tx pgx.Tx, hostID int64, entry HostSoftwareEntry) error {
	if entry.Name == "" || entry.Source == "" {
		return nil
	}
	softwareID, err := softwareIDFor(ctx, tx, entry)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO host_software (host_id, software_id, last_seen_at, last_opened_at)
VALUES ($1, $2, now(), $3)
ON CONFLICT (host_id, software_id) DO UPDATE SET
    last_seen_at = now(),
    last_opened_at = EXCLUDED.last_opened_at`,
		hostID, softwareID, entry.LastOpenedAt,
	); err != nil {
		return err
	}
	if entry.InstalledPath == "" {
		return nil
	}
	_, err = tx.Exec(ctx, `
INSERT INTO host_software_installed_paths (
    host_id, software_id, installed_path, team_identifier,
    cdhash_sha256, executable_sha256, executable_path, last_seen_at
)
VALUES (
    $1, $2, $3, $4,
    NULLIF($5::text, ''), NULLIF($6::text, ''), NULLIF($7::text, ''), now()
)
ON CONFLICT (host_id, software_id, installed_path) DO UPDATE SET
    team_identifier = EXCLUDED.team_identifier,
    cdhash_sha256 = EXCLUDED.cdhash_sha256,
    executable_sha256 = EXCLUDED.executable_sha256,
    executable_path = EXCLUDED.executable_path,
    last_seen_at = now()`,
		hostID, softwareID,
		entry.InstalledPath, entry.TeamIdentifier,
		entry.CDHashSHA256, entry.ExecutableSHA256, entry.ExecutablePath,
	)
	return err
}

func softwareIDFor(ctx context.Context, tx pgx.Tx, entry HostSoftwareEntry) (int64, error) {
	titleID, err := softwareTitleIDFor(ctx, tx, entry)
	if err != nil {
		return 0, err
	}
	var id int64
	err = tx.QueryRow(ctx, `
INSERT INTO software (
    title_id, name, version, source,
    bundle_identifier, extension_id, extension_for,
    vendor, arch, release
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (
    title_id, version, source,
    bundle_identifier, extension_id, extension_for,
    vendor, arch, release
) DO UPDATE SET updated_at = now()
RETURNING id`,
		titleID,
		entry.Name, entry.Version, entry.Source,
		entry.BundleIdentifier, entry.ExtensionID, entry.ExtensionFor,
		entry.Vendor, entry.Arch, entry.Release,
	).Scan(&id)
	return id, err
}

func softwareTitleIDFor(ctx context.Context, tx pgx.Tx, entry HostSoftwareEntry) (int64, error) {
	if entry.BundleIdentifier != "" {
		return upsertSoftwareTitleByBundle(ctx, tx, entry)
	}
	return upsertSoftwareTitleByName(ctx, tx, entry)
}

func upsertSoftwareTitleByBundle(ctx context.Context, tx pgx.Tx, entry HostSoftwareEntry) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
INSERT INTO software_titles (name, display_name, source, extension_for, bundle_identifier, vendor)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (bundle_identifier, source, extension_for)
WHERE bundle_identifier <> ''
DO UPDATE SET
    vendor = COALESCE(NULLIF(EXCLUDED.vendor, ''), software_titles.vendor),
    updated_at = now()
RETURNING id`,
		entry.Name, entry.Name, entry.Source, entry.ExtensionFor, entry.BundleIdentifier, entry.Vendor,
	).Scan(&id)
	return id, err
}

func upsertSoftwareTitleByName(ctx context.Context, tx pgx.Tx, entry HostSoftwareEntry) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
INSERT INTO software_titles (name, display_name, source, extension_for, bundle_identifier, vendor)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (name, source, extension_for, bundle_identifier) DO UPDATE SET
    vendor = COALESCE(NULLIF(EXCLUDED.vendor, ''), software_titles.vendor),
    updated_at = now()
RETURNING id`,
		entry.Name, entry.Name, entry.Source, entry.ExtensionFor, entry.BundleIdentifier, entry.Vendor,
	).Scan(&id)
	return id, err
}
