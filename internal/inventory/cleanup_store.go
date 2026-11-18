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
	versions, err := s.pruneOrphans(ctx, "software",
		`SELECT id FROM software WHERE NOT EXISTS (SELECT FROM host_software WHERE software_id = software.id)`)
	if err != nil {
		return CleanupResult{}, err
	}
	titles, err := s.pruneOrphans(ctx, "software_titles",
		`SELECT id FROM software_titles WHERE NOT EXISTS (SELECT FROM software WHERE title_id = software_titles.id)`)
	return CleanupResult{SoftwareVersions: versions, SoftwareTitles: titles}, err
}

func (s *Store) pruneOrphans(ctx context.Context, table, orphans string) (int64, error) {
	var after, total int64
	for {
		var ids []int64
		if err := s.pool.QueryRow(ctx, `SELECT ARRAY(`+orphans+` AND id > $1 ORDER BY id LIMIT $2)`, after, cleanupBatchSize).Scan(&ids); err != nil {
			return total, err
		}
		if len(ids) == 0 {
			return total, nil
		}
		n, err := s.pruneOrphanBatch(ctx, table, orphans, ids)
		if err != nil {
			return total, err
		}
		total += n
		if len(ids) < cleanupBatchSize {
			return total, nil
		}
		after = ids[len(ids)-1]
	}
}

func (s *Store) pruneOrphanBatch(ctx context.Context, table, orphans string, ids []int64) (int64, error) {
	var deleted int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Projections insert titles before resolving IDs. Exclude them only
		// while rechecking and deleting the selected batch.
		if _, err := tx.Exec(ctx, `LOCK TABLE software_titles IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
WITH candidates AS (`+orphans+` AND id = ANY($1) ORDER BY id FOR UPDATE SKIP LOCKED)
DELETE FROM `+table+` WHERE id IN (SELECT id FROM candidates)`, ids)
		if err == nil {
			deleted = tag.RowsAffected()
		}
		return err
	})
	return deleted, err
}
