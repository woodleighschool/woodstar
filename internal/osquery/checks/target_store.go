package checks

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
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
	qrows, err := s.pool.Query(ctx, listCheckTargetsSQL, checkIDs)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[checkTargetRow])
	if err != nil {
		return nil, err
	}
	targetSets, err := checkTargetSets(checkIDs, rows)
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
	rows := checkTargetWrites(checkID, targeting.LabelSet(targets))
	if err := postgres.ReplaceChildren(
		ctx, tx,
		deleteCheckTargetsSQL, []any{checkID},
		insertCheckTargetSQL, rows,
	); err != nil {
		return postgres.MutationError(err)
	}
	return nil
}

func (targets CheckTargets) validate() error {
	if err := targeting.ValidateLabelSets(targets.Include, targets.Exclude); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

func normalizeCheckTargets(targets CheckTargets) CheckTargets {
	return CheckTargets(targeting.NormalizeLabelSet(targeting.LabelSet(targets)))
}

func emptyCheckTargets() CheckTargets {
	return CheckTargets(targeting.EmptyLabelSet())
}

type checkTargetRow struct {
	OwnerID   int64  `db:"owner_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
}

type checkTargetWrite struct {
	OwnerID   int64  `db:"owner_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
	Position  int32  `db:"position"`
}

func checkTargetSets(ownerIDs []int64, rows []checkTargetRow) (map[int64]targeting.LabelSet, error) {
	targets := make(map[int64]targeting.LabelSet, len(ownerIDs))
	for _, ownerID := range ownerIDs {
		targets[ownerID] = targeting.EmptyLabelSet()
	}
	for _, row := range rows {
		targetSet := targets[row.OwnerID]
		ref := targeting.LabelRef{LabelID: row.LabelID}
		switch targeting.Direction(row.Direction) {
		case targeting.Include:
			targetSet.Include = append(targetSet.Include, ref)
		case targeting.Exclude:
			targetSet.Exclude = append(targetSet.Exclude, ref)
		default:
			return nil, fmt.Errorf("checks: unsupported target direction %q", row.Direction)
		}
		targets[row.OwnerID] = targetSet
	}
	return targets, nil
}

func checkTargetWrites(ownerID int64, targets targeting.LabelSet) []checkTargetWrite {
	targets = targeting.NormalizeLabelSet(targets)
	rows := make([]checkTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, ref := range targets.Include {
		rows = append(rows, checkTargetWrite{
			OwnerID:   ownerID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Include),
			Position:  int32(i),
		})
	}
	for i, ref := range targets.Exclude {
		rows = append(rows, checkTargetWrite{
			OwnerID:   ownerID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Exclude),
			Position:  int32(i),
		})
	}
	return rows
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
