package events

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const upsertBundleSQL = `
INSERT INTO santa_bundles AS b (
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
	bundle_id = COALESCE(NULLIF(EXCLUDED.bundle_id, ''), b.bundle_id),
	name = COALESCE(NULLIF(EXCLUDED.name, ''), b.name),
	path = COALESCE(NULLIF(EXCLUDED.path, ''), b.path),
	executable_rel_path = COALESCE(NULLIF(EXCLUDED.executable_rel_path, ''), b.executable_rel_path),
	version = COALESCE(NULLIF(EXCLUDED.version, ''), b.version),
	version_string = COALESCE(NULLIF(EXCLUDED.version_string, ''), b.version_string),
	binary_count = COALESCE(NULLIF(EXCLUDED.binary_count, 0), b.binary_count),
	hash_millis = COALESCE(NULLIF(EXCLUDED.hash_millis, 0), b.hash_millis),
	updated_at = now()
WHERE EXCLUDED.bundle_id NOT IN ('', b.bundle_id)
	OR EXCLUDED.name NOT IN ('', b.name)
	OR EXCLUDED.path NOT IN ('', b.path)
	OR EXCLUDED.executable_rel_path NOT IN ('', b.executable_rel_path)
	OR EXCLUDED.version NOT IN ('', b.version)
	OR EXCLUDED.version_string NOT IN ('', b.version_string)
	OR EXCLUDED.binary_count NOT IN (0, b.binary_count)
	OR EXCLUDED.hash_millis NOT IN (0, b.hash_millis)`

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

func reconcileBundleCompletion(ctx context.Context, tx pgx.Tx, bundleIDs []int64) error {
	if len(bundleIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
WITH completion AS (
	SELECT b.id, b.binary_count > 0 AND count(be.executable_id) >= b.binary_count AS complete
	FROM santa_bundles b
	LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
	WHERE b.id = ANY($1::bigint[])
	GROUP BY b.id
)
UPDATE santa_bundles b
SET uploaded_at = CASE WHEN c.complete THEN now() END, updated_at = now()
FROM completion c
WHERE b.id = c.id AND (b.uploaded_at IS NOT NULL) <> c.complete`, bundleIDs)
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

// merge retains the last non-empty value for each bundle field, matching the
// existing ON CONFLICT merge rules while folding duplicate references.
func (w *bundleWrite) merge(event ExecutionEventInput) {
	w.SHA256 = event.BundleHash
	if event.BundleID != "" {
		w.BundleID = event.BundleID
	}
	if event.BundleName != "" {
		w.Name = event.BundleName
	}
	if event.BundlePath != "" {
		w.Path = event.BundlePath
	}
	if event.BundleExecutableRelPath != "" {
		w.ExecutableRelPath = event.BundleExecutableRelPath
	}
	if event.BundleVersion != "" {
		w.Version = event.BundleVersion
	}
	if event.BundleVersionString != "" {
		w.VersionString = event.BundleVersionString
	}
	if event.BundleBinaryCount > 0 {
		w.BinaryCount = event.BundleBinaryCount
	}
	if event.BundleHashMillis > 0 {
		w.HashMillis = event.BundleHashMillis
	}
}
