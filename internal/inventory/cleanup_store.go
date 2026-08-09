package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const cleanupBatchSize = 1000

// CleanupResult reports how many unreferenced inventory rows were removed.
type CleanupResult struct {
	SoftwareVersions int64
	SoftwareTitles   int64
}

// PruneUnreferencedSoftware removes software versions without host references,
// followed by titles without remaining versions.
func (s *Store) PruneUnreferencedSoftware(ctx context.Context) (CleanupResult, error) {
	var total CleanupResult
	for {
		batch, err := s.pruneUnreferencedSoftwareBatch(ctx)
		if err != nil {
			return CleanupResult{}, err
		}
		total.SoftwareVersions += batch.SoftwareVersions
		total.SoftwareTitles += batch.SoftwareTitles
		if batch.SoftwareVersions < cleanupBatchSize && batch.SoftwareTitles < cleanupBatchSize {
			return total, nil
		}
	}
}

func (s *Store) pruneUnreferencedSoftwareBatch(ctx context.Context) (CleanupResult, error) {
	var result CleanupResult
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		softwareTag, err := tx.Exec(ctx, `
WITH candidates AS (
	SELECT software.id
	FROM software
	WHERE NOT EXISTS (
		SELECT 1
		FROM host_software
		WHERE host_software.software_id = software.id
	)
	ORDER BY software.id
	LIMIT $1
	FOR UPDATE OF software SKIP LOCKED
)
DELETE FROM software
USING candidates
WHERE software.id = candidates.id`, cleanupBatchSize)
		if err != nil {
			return err
		}
		result.SoftwareVersions = softwareTag.RowsAffected()

		titleTag, err := tx.Exec(ctx, `
WITH candidates AS (
	SELECT software_titles.id
	FROM software_titles
	WHERE NOT EXISTS (
		SELECT 1
		FROM software
		WHERE software.title_id = software_titles.id
	)
	ORDER BY software_titles.id
	LIMIT $1
	FOR UPDATE OF software_titles SKIP LOCKED
)
DELETE FROM software_titles
USING candidates
WHERE software_titles.id = candidates.id`, cleanupBatchSize)
		if err != nil {
			return err
		}
		result.SoftwareTitles = titleTag.RowsAffected()
		return nil
	})
	if err != nil {
		return CleanupResult{}, err
	}
	return result, nil
}
