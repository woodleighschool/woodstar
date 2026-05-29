package reports

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
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

	rows, err := s.q.ListReportScopes(ctx, sqlc.ListReportScopesParams{ReportIds: reportIDs})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		scopes[row.ID] = scope.LabelScope{Mode: row.LabelScopeMode}
	}

	labels, err := s.q.ListReportLabelIDs(ctx, sqlc.ListReportLabelIDsParams{ReportIds: reportIDs})
	if err != nil {
		return nil, err
	}
	for _, row := range labels {
		lscope := scopes[row.ReportID]
		lscope.LabelIDs = append(lscope.LabelIDs, row.LabelID)
		scopes[row.ReportID] = lscope
	}
	for reportID, lscope := range scopes {
		scopes[reportID] = scope.NormalizeLabelScope(lscope)
	}
	return scopes, nil
}

func replaceReportScope(ctx context.Context, tx pgx.Tx, reportID int64, lscope scope.LabelScope) error {
	lscope = scope.NormalizeLabelScope(lscope)
	q := sqlc.New(tx)
	if err := q.SetReportScopeMode(ctx, sqlc.SetReportScopeModeParams{
		ID:             reportID,
		LabelScopeMode: lscope.Mode,
	}); err != nil {
		return err
	}
	if err := q.DeleteReportLabels(ctx, sqlc.DeleteReportLabelsParams{ReportID: reportID}); err != nil {
		return err
	}
	if len(lscope.LabelIDs) == 0 {
		return nil
	}
	return q.InsertReportLabels(ctx, sqlc.InsertReportLabelsParams{
		ReportID: reportID,
		LabelIds: lscope.LabelIDs,
	})
}
