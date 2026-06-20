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

type checkTargetRow struct {
	CheckID   int64  `db:"check_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
}

func (s *Store) loadCheckTargets(
	ctx context.Context,
	checkIDs []int64,
) (map[int64]CheckTargets, error) {
	targets := make(map[int64]CheckTargets, len(checkIDs))
	if len(checkIDs) == 0 {
		return targets, nil
	}
	for _, checkID := range checkIDs {
		targets[checkID] = emptyCheckTargets()
	}
	rows, err := s.db.Pool().Query(ctx, listCheckTargetsSQL, checkIDs)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[checkTargetRow])
	if err != nil {
		return nil, err
	}
	for _, row := range records {
		targetSet := targets[row.CheckID]
		ref := targeting.LabelRef{LabelID: row.LabelID}
		switch targeting.Direction(row.Direction) {
		case targeting.Include:
			targetSet.Include = append(targetSet.Include, ref)
		case targeting.Exclude:
			targetSet.Exclude = append(targetSet.Exclude, ref)
		default:
			return nil, fmt.Errorf("%w: unsupported target direction %q", dbutil.ErrInvalidInput, row.Direction)
		}
		targets[row.CheckID] = targetSet
	}
	return targets, nil
}

type checkTargetWrite struct {
	CheckID   int64  `db:"check_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
	Position  int32  `db:"position"`
}

const insertCheckTargetSQL = `
INSERT INTO osquery_check_targets (check_id, label_id, direction, position)
VALUES (@check_id, @label_id, @direction::target_direction, @position)`

func replaceCheckTargets(ctx context.Context, tx pgx.Tx, checkID int64, targets CheckTargets) error {
	targets = normalizeCheckTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	rows := make([]checkTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, ref := range targets.Include {
		rows = append(rows, checkTargetWrite{
			CheckID:   checkID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Include),
			Position:  int32(i),
		})
	}
	for i, ref := range targets.Exclude {
		rows = append(rows, checkTargetWrite{
			CheckID:   checkID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Exclude),
			Position:  int32(i),
		})
	}
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
	if targets.Include == nil {
		targets.Include = []targeting.LabelRef{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []targeting.LabelRef{}
	}
	return targets
}

func emptyCheckTargets() CheckTargets {
	return CheckTargets{
		Include: []targeting.LabelRef{},
		Exclude: []targeting.LabelRef{},
	}
}

const listCheckTargetsSQL = `
SELECT check_id, label_id, direction::text AS direction
FROM osquery_check_targets
WHERE check_id = ANY($1::bigint[])
ORDER BY
    check_id,
    direction,
    position`

const deleteCheckTargetsSQL = `DELETE FROM osquery_check_targets WHERE check_id = $1`
