package reports

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// ReportTargets is the include/exclude label targeting contract for a report.
type ReportTargets struct {
	Include []targeting.LabelRef `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

func (s *Store) loadReportTarget(ctx context.Context, reportID int64) (ReportTargets, error) {
	targets, err := s.loadReportTargets(ctx, []int64{reportID})
	if err != nil {
		return ReportTargets{}, err
	}
	if rows, ok := targets[reportID]; ok {
		return rows, nil
	}
	return emptyReportTargets(), nil
}

func (s *Store) loadReportTargets(
	ctx context.Context,
	reportIDs []int64,
) (map[int64]ReportTargets, error) {
	targets := make(map[int64]ReportTargets, len(reportIDs))
	if len(reportIDs) == 0 {
		return targets, nil
	}
	for _, reportID := range reportIDs {
		targets[reportID] = emptyReportTargets()
	}
	rows, err := s.q.ListReportTargets(ctx, sqlc.ListReportTargetsParams{ReportIds: reportIDs})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		targetSet := targets[row.ReportID]
		ref := targeting.LabelRef{LabelID: row.LabelID}
		switch targeting.Direction(row.Direction) {
		case targeting.Include:
			targetSet.Include = append(targetSet.Include, ref)
		case targeting.Exclude:
			targetSet.Exclude = append(targetSet.Exclude, ref)
		default:
			return nil, fmt.Errorf("%w: unsupported target direction %q", dbutil.ErrInvalidInput, row.Direction)
		}
		targets[row.ReportID] = targetSet
	}
	return targets, nil
}

func replaceReportTargets(ctx context.Context, tx pgx.Tx, reportID int64, targets ReportTargets) error {
	targets = normalizeReportTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	q := sqlc.New(tx)
	if err := q.DeleteReportTargets(ctx, sqlc.DeleteReportTargetsParams{ReportID: reportID}); err != nil {
		return err
	}
	if len(targets.Include) > 0 {
		if err := q.InsertReportTargets(ctx, sqlc.InsertReportTargetsParams{
			ReportID:  reportID,
			LabelIds:  targeting.LabelRefIDs(targets.Include),
			Direction: sqlc.TargetDirection(targeting.Include),
		}); err != nil {
			return err
		}
	}
	if len(targets.Exclude) > 0 {
		if err := q.InsertReportTargets(ctx, sqlc.InsertReportTargetsParams{
			ReportID:  reportID,
			LabelIds:  targeting.LabelRefIDs(targets.Exclude),
			Direction: sqlc.TargetDirection(targeting.Exclude),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (targets ReportTargets) validate() error {
	if err := targeting.ValidateLabelSets(targets.Include, targets.Exclude); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func normalizeReportTargets(targets ReportTargets) ReportTargets {
	if targets.Include == nil {
		targets.Include = []targeting.LabelRef{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []targeting.LabelRef{}
	}
	return targets
}

func emptyReportTargets() ReportTargets {
	return ReportTargets{
		Include: []targeting.LabelRef{},
		Exclude: []targeting.LabelRef{},
	}
}
