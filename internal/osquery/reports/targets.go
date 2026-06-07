package reports

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func (s *Store) loadReportTarget(ctx context.Context, reportID int64) ([]scope.TargetLabel, error) {
	targets, err := s.loadReportTargets(ctx, []int64{reportID})
	if err != nil {
		return nil, err
	}
	if rows, ok := targets[reportID]; ok {
		return rows, nil
	}
	return []scope.TargetLabel{}, nil
}

func (s *Store) loadReportTargets(
	ctx context.Context,
	reportIDs []int64,
) (map[int64][]scope.TargetLabel, error) {
	if len(reportIDs) == 0 {
		return map[int64][]scope.TargetLabel{}, nil
	}
	rows, err := s.q.ListReportTargets(ctx, sqlc.ListReportTargetsParams{ReportIds: reportIDs})
	if err != nil {
		return nil, err
	}
	targets := make(map[int64][]scope.TargetLabel, len(reportIDs))
	for _, row := range rows {
		targets[row.ReportID] = append(targets[row.ReportID], scope.TargetLabel{
			LabelID: row.LabelID,
			Effect:  scope.TargetLabelEffect(row.Effect),
		})
	}
	return targets, nil
}

func replaceReportTargets(ctx context.Context, tx pgx.Tx, reportID int64, targets []scope.TargetLabel) error {
	if err := validateTargets(targets); err != nil {
		return err
	}
	q := sqlc.New(tx)
	if err := q.DeleteReportTargets(ctx, sqlc.DeleteReportTargetsParams{ReportID: reportID}); err != nil {
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
	return q.InsertReportTargets(ctx, sqlc.InsertReportTargetsParams{
		ReportID: reportID,
		LabelIds: labelIDs,
		Effects:  effects,
	})
}

func validateTargets(targets []scope.TargetLabel) error {
	seen := make(map[int64]struct{}, len(targets))
	for _, target := range targets {
		if !scope.ValidTargetLabelEffect(target.Effect) {
			return fmt.Errorf("%w: unsupported target effect %q", dbutil.ErrInvalidInput, target.Effect)
		}
		if _, ok := seen[target.LabelID]; ok {
			return fmt.Errorf("%w: duplicate target row", dbutil.ErrInvalidInput)
		}
		seen[target.LabelID] = struct{}{}
	}
	return nil
}
