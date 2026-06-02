package checks

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func (s *Store) loadCheckTarget(ctx context.Context, checkID int64) ([]scope.TargetLabel, error) {
	targets, err := s.loadCheckTargets(ctx, []int64{checkID})
	if err != nil {
		return nil, err
	}
	if rows, ok := targets[checkID]; ok {
		return rows, nil
	}
	return []scope.TargetLabel{}, nil
}

func (s *Store) loadCheckTargets(
	ctx context.Context,
	checkIDs []int64,
) (map[int64][]scope.TargetLabel, error) {
	if len(checkIDs) == 0 {
		return map[int64][]scope.TargetLabel{}, nil
	}
	rows, err := s.q.ListCheckTargets(ctx, sqlc.ListCheckTargetsParams{CheckIds: checkIDs})
	if err != nil {
		return nil, err
	}
	targets := make(map[int64][]scope.TargetLabel, len(checkIDs))
	for _, row := range rows {
		targets[row.CheckID] = append(targets[row.CheckID], scope.TargetLabel{
			LabelID: row.LabelID,
			Effect:  scope.TargetLabelEffect(row.Effect),
		})
	}
	return targets, nil
}

func replaceCheckTargets(ctx context.Context, tx pgx.Tx, checkID int64, targets []scope.TargetLabel) error {
	if err := validateTargets(targets); err != nil {
		return err
	}
	q := sqlc.New(tx)
	if err := q.DeleteCheckTargets(ctx, sqlc.DeleteCheckTargetsParams{CheckID: checkID}); err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}
	labelIDs := make([]int64, len(targets))
	effects := make([]string, len(targets))
	for i, target := range targets {
		labelIDs[i] = target.LabelID
		effects[i] = string(target.Effect)
	}
	return q.InsertCheckTargets(ctx, sqlc.InsertCheckTargetsParams{
		CheckID:  checkID,
		LabelIds: labelIDs,
		Effects:  effects,
	})
}

func validateTargets(targets []scope.TargetLabel) error {
	seen := make(map[scope.TargetLabel]struct{}, len(targets))
	for _, target := range targets {
		if target.LabelID <= 0 {
			return fmt.Errorf("%w: target label_id must be positive", dbutil.ErrInvalidInput)
		}
		if !scope.ValidTargetLabelEffect(target.Effect) {
			return fmt.Errorf("%w: unsupported target effect %q", dbutil.ErrInvalidInput, target.Effect)
		}
		if _, ok := seen[target]; ok {
			return fmt.Errorf("%w: duplicate target row", dbutil.ErrInvalidInput)
		}
		seen[target] = struct{}{}
	}
	return nil
}
