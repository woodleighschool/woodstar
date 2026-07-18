package packages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// ObjectPrefix namespaces package installer objects in storage.
const ObjectPrefix = "munki/packages"

const detachedObjectCleanupTimeout = 15 * time.Second

type objectStore interface {
	Delete(ctx context.Context, objectID int64) error
}

func validateAndLockInstallerObject(
	ctx context.Context,
	tx pgx.Tx,
	objectID *int64,
	packageID int64,
) error {
	if objectID == nil {
		return nil
	}
	var prefix string
	var sizeBytes *int64
	var sha256sum *string
	var availableAt *time.Time
	if err := tx.QueryRow(ctx, `
SELECT prefix, size_bytes, sha256, available_at
FROM storage_objects
WHERE id = $1
FOR UPDATE`, *objectID).Scan(&prefix, &sizeBytes, &sha256sum, &availableAt); err != nil {
		return dbutil.GetError(err)
	}
	if prefix != ObjectPrefix {
		return fmt.Errorf("%w: installer_object_id must reference a package installer", dbutil.ErrInvalidInput)
	}
	if availableAt == nil || sizeBytes == nil || sha256sum == nil {
		return fmt.Errorf("%w: installer_object_id must reference a finalized object", dbutil.ErrInvalidInput)
	}
	var ownerID int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM munki_packages
WHERE installer_object_id = $1
  AND id <> $2
LIMIT 1`, *objectID, packageID).Scan(&ownerID)
	if err == nil {
		return fmt.Errorf("%w: installer object is already owned by package %d", dbutil.ErrConflict, ownerID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}

func deleteObjects(ctx context.Context, objects objectStore, ids ...int64) error {
	for _, id := range ids {
		if err := objects.Delete(ctx, id); err != nil &&
			!errors.Is(err, dbutil.ErrConflict) &&
			!errors.Is(err, dbutil.ErrNotFound) {
			return err
		}
	}
	return nil
}

func replacedObjectID(oldID, newID *int64) []int64 {
	if oldID == nil || newID != nil && *oldID == *newID {
		return nil
	}
	return []int64{*oldID}
}

func (s *Store) packageObjectIDs(ctx context.Context, q dbutil.Queryer, ids []int64) ([]int64, error) {
	rows, err := q.Query(ctx, `
		SELECT refs.object_id::bigint AS object_id
		FROM munki_packages p
		CROSS JOIN LATERAL unnest(array_remove(ARRAY[p.installer_object_id], NULL)::bigint[]) AS refs(object_id)
		WHERE p.id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}
