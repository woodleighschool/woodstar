package reports

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/osquery/labelscope"
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
	if len(reportIDs) == 0 {
		return map[int64]scope.LabelScope{}, nil
	}
	return labelscope.LoadReports(ctx, s.q, reportIDs)
}

func replaceReportScope(ctx context.Context, tx pgx.Tx, reportID int64, lscope scope.LabelScope) error {
	return labelscope.ReplaceReport(ctx, tx, reportID, lscope)
}

func storedLabelScope(lscope scope.LabelScope) scope.LabelScope {
	return labelscope.Normalize(lscope)
}
