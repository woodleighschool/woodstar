package checks

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// CheckTargets is the include/exclude label targeting contract for a check.
type CheckTargets struct {
	Include []targeting.LabelRef `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

func (s *Store) loadCheckTarget(ctx context.Context, checkID int64) (CheckTargets, error) {
	targets, err := s.loadCheckTargets(ctx, []int64{checkID})
	if err != nil {
		return CheckTargets{}, err
	}
	if rows, ok := targets[checkID]; ok {
		return rows, nil
	}
	return emptyCheckTargets(), nil
}

func (s *Store) loadCheckTargets(
	ctx context.Context,
	checkIDs []int64,
) (map[int64]CheckTargets, error) {
	if len(checkIDs) == 0 {
		return map[int64]CheckTargets{}, nil
	}
	rows, err := s.db.Pool().Query(ctx, listCheckTargetsSQL, checkIDs)
	if err != nil {
		return nil, err
	}
	records, err := targeting.CollectLabelTargetRows(rows)
	if err != nil {
		return nil, err
	}
	targetSets, err := targeting.LabelSetsFromRows(checkIDs, records)
	if err != nil {
		return nil, err
	}
	targets := make(map[int64]CheckTargets, len(targetSets))
	for checkID, targetSet := range targetSets {
		targets[checkID] = CheckTargets(targetSet)
	}
	return targets, nil
}

const insertCheckTargetSQL = `
INSERT INTO osquery_check_targets (check_id, label_id, direction, position)
VALUES (@owner_id, @label_id, @direction::target_direction, @position)`

func replaceCheckTargets(ctx context.Context, tx pgx.Tx, checkID int64, targets CheckTargets) error {
	targets = normalizeCheckTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	rows := targeting.LabelTargetWrites(checkID, targeting.LabelSet(targets))
	if err := dbutil.ReplaceChildren(
		ctx, tx,
		deleteCheckTargetsSQL, []any{checkID},
		insertCheckTargetSQL, rows,
	); err != nil {
		return dbutil.MutationError(err)
	}
	return nil
}

func (targets CheckTargets) validate() error {
	if err := targeting.ValidateLabelSets(targets.Include, targets.Exclude); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func normalizeCheckTargets(targets CheckTargets) CheckTargets {
	return CheckTargets(targeting.NormalizeLabelSet(targeting.LabelSet(targets)))
}

func emptyCheckTargets() CheckTargets {
	return CheckTargets(targeting.EmptyLabelSet())
}

const listCheckTargetsSQL = `
SELECT check_id AS owner_id, label_id, direction::text AS direction
FROM osquery_check_targets
WHERE check_id = ANY($1::bigint[])
ORDER BY
    check_id,
    direction,
    position`

const deleteCheckTargetsSQL = `DELETE FROM osquery_check_targets WHERE check_id = $1`
