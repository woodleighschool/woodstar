package reports

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// ReportTargets is the include/exclude label targeting contract for a report.
type ReportTargets struct {
	Include []targeting.LabelRef `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

func (s *Store) attachReportTargets(
	ctx context.Context,
	rpts []Report,
	reportIDs []int64,
) error {
	if len(reportIDs) == 0 {
		return nil
	}
	rptIndexes := make(map[int64]int, len(rpts))
	for i := range rpts {
		rptIndexes[rpts[i].ID] = i
		rpts[i].Targets = emptyReportTargets()
	}

	qrows, err := s.db.Pool().Query(ctx, `
		SELECT report_id AS owner_id, label_id, direction::text AS direction
		FROM osquery_report_targets
		WHERE report_id = ANY($1::bigint[])
		ORDER BY report_id, direction, position`,
		reportIDs,
	)
	if err != nil {
		return err
	}
	rows, err := targeting.CollectLabelTargetRows(qrows)
	if err != nil {
		return err
	}
	targets, err := targeting.LabelSetsFromRows(reportIDs, rows)
	if err != nil {
		return err
	}
	for reportID, targetSet := range targets {
		if i, ok := rptIndexes[reportID]; ok {
			rpts[i].Targets = ReportTargets(targetSet)
		}
	}
	return nil
}

const deleteReportTargetsSQL = `DELETE FROM osquery_report_targets WHERE report_id = $1`

const insertReportTargetSQL = `
INSERT INTO osquery_report_targets (report_id, label_id, direction, position)
VALUES (@owner_id, @label_id, @direction::target_direction, @position)`

func replaceReportTargets(ctx context.Context, tx pgx.Tx, reportID int64, targets ReportTargets) error {
	targets = normalizeReportTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	rows := targeting.LabelTargetWrites(reportID, targeting.LabelSet(targets))
	if err := dbutil.ReplaceChildren(
		ctx, tx,
		deleteReportTargetsSQL, []any{reportID},
		insertReportTargetSQL, rows,
	); err != nil {
		return dbutil.MutationError(err)
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
	return ReportTargets(targeting.NormalizeLabelSet(targeting.LabelSet(targets)))
}

func emptyReportTargets() ReportTargets {
	return ReportTargets(targeting.EmptyLabelSet())
}
