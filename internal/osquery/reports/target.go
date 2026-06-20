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

	type targetRow struct {
		ReportID  int64  `db:"report_id"`
		LabelID   int64  `db:"label_id"`
		Direction string `db:"direction"`
	}

	qrows, err := s.db.Pool().Query(ctx, `
		SELECT report_id, label_id, direction::text AS direction
		FROM osquery_report_targets
		WHERE report_id = ANY($1::bigint[])
		ORDER BY report_id, direction, position`,
		reportIDs,
	)
	if err != nil {
		return err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[targetRow])
	if err != nil {
		return err
	}
	for _, row := range rows {
		if i, ok := rptIndexes[row.ReportID]; ok {
			targetSet := rpts[i].Targets
			ref := targeting.LabelRef{LabelID: row.LabelID}
			switch targeting.Direction(row.Direction) {
			case targeting.Include:
				targetSet.Include = append(targetSet.Include, ref)
			case targeting.Exclude:
				targetSet.Exclude = append(targetSet.Exclude, ref)
			default:
				return fmt.Errorf("%w: unsupported target direction %q", dbutil.ErrInvalidInput, row.Direction)
			}
			rpts[i].Targets = targetSet
		}
	}
	return nil
}

type reportTargetWrite struct {
	ReportID  int64  `db:"report_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
	Position  int32  `db:"position"`
}

const deleteReportTargetsSQL = `DELETE FROM osquery_report_targets WHERE report_id = $1`

const insertReportTargetSQL = `
INSERT INTO osquery_report_targets (report_id, label_id, direction, position)
VALUES (@report_id, @label_id, @direction::target_direction, @position)`

func replaceReportTargets(ctx context.Context, tx pgx.Tx, reportID int64, targets ReportTargets) error {
	targets = normalizeReportTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	rows := make([]reportTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, ref := range targets.Include {
		rows = append(rows, reportTargetWrite{
			ReportID:  reportID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Include),
			Position:  int32(i),
		})
	}
	for i, ref := range targets.Exclude {
		rows = append(rows, reportTargetWrite{
			ReportID:  reportID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Exclude),
			Position:  int32(i),
		})
	}
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
