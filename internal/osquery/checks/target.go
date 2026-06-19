package checks

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
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
	targets := make(map[int64]CheckTargets, len(checkIDs))
	if len(checkIDs) == 0 {
		return targets, nil
	}
	for _, checkID := range checkIDs {
		targets[checkID] = emptyCheckTargets()
	}
	rows, err := s.q.ListCheckTargets(ctx, sqlc.ListCheckTargetsParams{CheckIds: checkIDs})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
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

func replaceCheckTargets(ctx context.Context, tx pgx.Tx, checkID int64, targets CheckTargets) error {
	targets = normalizeCheckTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	q := sqlc.New(tx)
	if err := q.DeleteCheckTargets(ctx, sqlc.DeleteCheckTargetsParams{CheckID: checkID}); err != nil {
		return err
	}
	if len(targets.Include) > 0 {
		if err := q.InsertCheckTargets(ctx, sqlc.InsertCheckTargetsParams{
			CheckID:   checkID,
			LabelIds:  targeting.LabelRefIDs(targets.Include),
			Direction: sqlc.TargetDirection(targeting.Include),
		}); err != nil {
			return err
		}
	}
	if len(targets.Exclude) > 0 {
		if err := q.InsertCheckTargets(ctx, sqlc.InsertCheckTargetsParams{
			CheckID:   checkID,
			LabelIds:  targeting.LabelRefIDs(targets.Exclude),
			Direction: sqlc.TargetDirection(targeting.Exclude),
		}); err != nil {
			return err
		}
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
