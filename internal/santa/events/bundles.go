// Package events persists and queries Santa execution and file-access events.
package events

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// processEventBundle upserts and links the event's bundle when present and
// returns the bundle hash to request a binary listing for, or "" when none.
func processEventBundle(
	ctx context.Context,
	tx pgx.Tx,
	executableID int64,
	event ExecutionEventInput,
) (string, error) {
	bundleID, hasBundle, err := upsertBundle(ctx, tx, event)
	if err != nil {
		return "", err
	}
	if !hasBundle {
		return "", nil
	}
	if err := linkBundleExecutable(ctx, tx, bundleID, executableID); err != nil {
		return "", err
	}
	if err := refreshBundleUploadedAt(ctx, tx, bundleID); err != nil {
		return "", err
	}
	if event.Decision == ExecutionDecisionBundleBinary {
		return "", nil
	}
	return event.BundleHash, nil
}

func upsertBundle(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, bool, error) {
	if event.BundleHash == "" {
		return 0, false, nil
	}
	write := bundleWrite{
		SHA256:            event.BundleHash,
		BundleID:          event.BundleID,
		Name:              event.BundleName,
		Path:              event.BundlePath,
		ExecutableRelPath: event.BundleExecutableRelPath,
		Version:           event.BundleVersion,
		VersionString:     event.BundleVersionString,
		BinaryCount:       event.BundleBinaryCount,
		HashMillis:        event.BundleHashMillis,
	}
	var id int64
	if err := tx.QueryRow(ctx, `
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
RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func linkBundleExecutable(ctx context.Context, tx pgx.Tx, bundleID int64, executableID int64) error {
	_, err := tx.Exec(ctx, `
INSERT INTO santa_bundle_executables (bundle_id, executable_id)
VALUES (@bundle_id, @executable_id)
ON CONFLICT DO NOTHING`,
		pgx.NamedArgs{
			"bundle_id":     bundleID,
			"executable_id": executableID,
		})
	return err
}

func refreshBundleUploadedAt(ctx context.Context, tx pgx.Tx, bundleID int64) error {
	_, err := tx.Exec(ctx, `
UPDATE santa_bundles b
SET uploaded_at = COALESCE(uploaded_at, now()), updated_at = now()
WHERE b.id = @bundle_id
  AND b.binary_count > 0
  AND (
	  SELECT count(*)
	  FROM santa_bundle_executables be
	  WHERE be.bundle_id = b.id
  ) >= b.binary_count`,
		pgx.NamedArgs{"bundle_id": bundleID},
	)
	return err
}

func incompleteBundleHashes(ctx context.Context, tx pgx.Tx, candidates []string) ([]string, error) {
	hashes := normalizeStringSlice(candidates)
	if len(hashes) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
SELECT b.sha256
FROM santa_bundles b
WHERE b.sha256 = ANY($1::text[])
  AND b.uploaded_at IS NULL
ORDER BY b.sha256`,
		hashes,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

type bundleWrite struct {
	SHA256            string `db:"sha256"`
	BundleID          string `db:"bundle_id"`
	Name              string `db:"name"`
	Path              string `db:"path"`
	ExecutableRelPath string `db:"executable_rel_path"`
	Version           string `db:"version"`
	VersionString     string `db:"version_string"`
	BinaryCount       uint32 `db:"binary_count"`
	HashMillis        uint32 `db:"hash_millis"`
}
