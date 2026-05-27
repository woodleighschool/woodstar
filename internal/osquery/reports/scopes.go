package reports

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func (s *Store) loadReportScope(ctx context.Context, reportID int64) (scope.LabelScope, error) {
	scopes, err := s.loadReportScopes(ctx, []int64{reportID})
	if err != nil {
		return scope.LabelScope{}, err
	}
	lscope, ok := scopes[reportID]
	if !ok {
		return scope.LabelScope{}, dbutil.ErrNotFound
	}
	return lscope, nil
}

func (s *Store) loadReportScopes(ctx context.Context, reportIDs []int64) (map[int64]scope.LabelScope, error) {
	scopes := make(map[int64]scope.LabelScope, len(reportIDs))
	if len(reportIDs) == 0 {
		return scopes, nil
	}

	rows, err := s.db.Pool().Query(ctx,
		`SELECT id, label_scope_mode FROM reports WHERE id = ANY($1::bigint[])`, reportIDs)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var reportID int64
		var mode scope.LabelScopeMode
		if err := rows.Scan(&reportID, &mode); err != nil {
			rows.Close()
			return nil, err
		}
		scopes[reportID] = scope.LabelScope{Mode: mode}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = s.db.Pool().Query(ctx,
		`SELECT report_id, label_id FROM report_labels WHERE report_id = ANY($1::bigint[]) ORDER BY report_id, label_id`,
		reportIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var reportID int64
		var labelID int64
		if err := rows.Scan(&reportID, &labelID); err != nil {
			return nil, err
		}
		lscope := scopes[reportID]
		lscope.LabelIDs = append(lscope.LabelIDs, labelID)
		scopes[reportID] = lscope
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for reportID, lscope := range scopes {
		scopes[reportID] = scope.NormalizeLabelScope(lscope)
	}
	return scopes, nil
}

func replaceReportScope(ctx context.Context, tx pgx.Tx, reportID int64, lscope scope.LabelScope) error {
	lscope = scope.NormalizeLabelScope(lscope)
	if _, err := tx.Exec(ctx, `UPDATE reports SET label_scope_mode = $2 WHERE id = $1`,
		reportID, string(lscope.Mode)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM report_labels WHERE report_id = $1`, reportID); err != nil {
		return err
	}
	if len(lscope.LabelIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx,
		`INSERT INTO report_labels (report_id, label_id) SELECT $1, unnest($2::bigint[])`,
		reportID, lscope.LabelIDs)
	return err
}
