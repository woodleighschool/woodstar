package reports

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
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

	qrows, err := s.pool.Query(ctx, `
		SELECT report_id AS owner_id, label_id, direction::text AS direction
		FROM osquery_report_targets
		WHERE report_id = ANY($1::bigint[])
		ORDER BY report_id, direction, position`,
		reportIDs,
	)
	if err != nil {
		return err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[reportTargetRow])
	if err != nil {
		return err
	}
	targets, err := reportTargetSets(reportIDs, rows)
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
	rows := reportTargetWrites(reportID, targeting.LabelSet(targets))
	if err := postgres.ReplaceChildren(
		ctx, tx,
		deleteReportTargetsSQL, []any{reportID},
		insertReportTargetSQL, rows,
	); err != nil {
		return postgres.MutationError(err)
	}
	return nil
}

func (targets ReportTargets) validate() error {
	if err := targeting.ValidateLabelSets(targets.Include, targets.Exclude); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

func normalizeReportTargets(targets ReportTargets) ReportTargets {
	return ReportTargets(targeting.NormalizeLabelSet(targeting.LabelSet(targets)))
}

func emptyReportTargets() ReportTargets {
	return ReportTargets(targeting.EmptyLabelSet())
}

type reportTargetRow struct {
	OwnerID   int64  `db:"owner_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
}

type reportTargetWrite struct {
	OwnerID   int64  `db:"owner_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
	Position  int32  `db:"position"`
}

func reportTargetSets(ownerIDs []int64, rows []reportTargetRow) (map[int64]targeting.LabelSet, error) {
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
			return nil, fmt.Errorf("reports: unsupported target direction %q", row.Direction)
		}
		targets[row.OwnerID] = targetSet
	}
	return targets, nil
}

func reportTargetWrites(ownerID int64, targets targeting.LabelSet) []reportTargetWrite {
	targets = targeting.NormalizeLabelSet(targets)
	rows := make([]reportTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, ref := range targets.Include {
		rows = append(rows, reportTargetWrite{
			OwnerID:   ownerID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Include),
			Position:  int32(i),
		})
	}
	for i, ref := range targets.Exclude {
		rows = append(rows, reportTargetWrite{
			OwnerID:   ownerID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Exclude),
			Position:  int32(i),
		})
	}
	return rows
}
