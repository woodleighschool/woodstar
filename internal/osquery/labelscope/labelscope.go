package labelscope

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/scope"
)

type scopeRow struct {
	ID   int64
	Mode scope.LabelScopeMode
}

type labelRow struct {
	OwnerID int64
	LabelID int64
}

func LoadChecks(
	ctx context.Context,
	q *sqlc.Queries,
	checkIDs []int64,
) (map[int64]scope.LabelScope, error) {
	rows, err := q.ListCheckScopes(ctx, sqlc.ListCheckScopesParams{CheckIds: checkIDs})
	if err != nil {
		return nil, err
	}
	labels, err := q.ListCheckLabelIDs(ctx, sqlc.ListCheckLabelIDsParams{CheckIds: checkIDs})
	if err != nil {
		return nil, err
	}
	scopeRows := make([]scopeRow, len(rows))
	for i, row := range rows {
		scopeRows[i] = scopeRow{ID: row.ID, Mode: row.LabelScopeMode}
	}
	labelRows := make([]labelRow, len(labels))
	for i, row := range labels {
		labelRows[i] = labelRow{OwnerID: row.CheckID, LabelID: row.LabelID}
	}
	return build(scopeRows, labelRows, len(checkIDs)), nil
}

func LoadReports(
	ctx context.Context,
	q *sqlc.Queries,
	reportIDs []int64,
) (map[int64]scope.LabelScope, error) {
	rows, err := q.ListReportScopes(ctx, sqlc.ListReportScopesParams{ReportIds: reportIDs})
	if err != nil {
		return nil, err
	}
	labels, err := q.ListReportLabelIDs(ctx, sqlc.ListReportLabelIDsParams{ReportIds: reportIDs})
	if err != nil {
		return nil, err
	}
	scopeRows := make([]scopeRow, len(rows))
	for i, row := range rows {
		scopeRows[i] = scopeRow{ID: row.ID, Mode: row.LabelScopeMode}
	}
	labelRows := make([]labelRow, len(labels))
	for i, row := range labels {
		labelRows[i] = labelRow{OwnerID: row.ReportID, LabelID: row.LabelID}
	}
	return build(scopeRows, labelRows, len(reportIDs)), nil
}

func ReplaceCheck(ctx context.Context, tx pgx.Tx, checkID int64, lscope scope.LabelScope) error {
	lscope = Normalize(lscope)
	q := sqlc.New(tx)
	if err := q.SetCheckScopeMode(ctx, sqlc.SetCheckScopeModeParams{
		ID:             checkID,
		LabelScopeMode: lscope.Mode,
	}); err != nil {
		return err
	}
	if err := q.DeleteCheckLabels(ctx, sqlc.DeleteCheckLabelsParams{CheckID: checkID}); err != nil {
		return err
	}
	if len(lscope.LabelIDs) == 0 {
		return nil
	}
	return q.InsertCheckLabels(ctx, sqlc.InsertCheckLabelsParams{
		CheckID:  checkID,
		LabelIds: lscope.LabelIDs,
	})
}

func ReplaceReport(ctx context.Context, tx pgx.Tx, reportID int64, lscope scope.LabelScope) error {
	lscope = Normalize(lscope)
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

func Normalize(lscope scope.LabelScope) scope.LabelScope {
	lscope = scope.NormalizeLabelScope(lscope)
	lscope.LabelIDs = uniqueInt64s(lscope.LabelIDs)
	return lscope
}

func build(
	scopeRows []scopeRow,
	labelRows []labelRow,
	capacity int,
) map[int64]scope.LabelScope {
	scopes := make(map[int64]scope.LabelScope, capacity)
	for _, row := range scopeRows {
		scopes[row.ID] = scope.LabelScope{Mode: row.Mode}
	}
	for _, row := range labelRows {
		lscope := scopes[row.OwnerID]
		lscope.LabelIDs = append(lscope.LabelIDs, row.LabelID)
		scopes[row.OwnerID] = lscope
	}
	for ownerID, lscope := range scopes {
		scopes[ownerID] = scope.NormalizeLabelScope(lscope)
	}
	return scopes
}

func uniqueInt64s(values []int64) []int64 {
	out := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
