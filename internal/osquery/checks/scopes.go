package checks

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func (s *Store) loadCheckScope(ctx context.Context, checkID int64) (scope.LabelScope, error) {
	scopes, err := s.loadCheckScopes(ctx, []int64{checkID})
	if err != nil {
		return scope.LabelScope{}, err
	}
	lscope, ok := scopes[checkID]
	if !ok {
		return scope.LabelScope{}, dbutil.ErrNotFound
	}
	return lscope, nil
}

func (s *Store) loadCheckScopes(ctx context.Context, checkIDs []int64) (map[int64]scope.LabelScope, error) {
	scopes := make(map[int64]scope.LabelScope, len(checkIDs))
	if len(checkIDs) == 0 {
		return scopes, nil
	}

	rows, err := s.q.ListCheckScopes(ctx, sqlc.ListCheckScopesParams{CheckIds: checkIDs})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		scopes[row.ID] = scope.LabelScope{Mode: row.LabelScopeMode}
	}

	labels, err := s.q.ListCheckLabelIDs(ctx, sqlc.ListCheckLabelIDsParams{CheckIds: checkIDs})
	if err != nil {
		return nil, err
	}
	for _, row := range labels {
		lscope := scopes[row.CheckID]
		lscope.LabelIDs = append(lscope.LabelIDs, row.LabelID)
		scopes[row.CheckID] = lscope
	}
	for checkID, lscope := range scopes {
		scopes[checkID] = scope.NormalizeLabelScope(lscope)
	}
	return scopes, nil
}

func replaceCheckScope(ctx context.Context, tx pgx.Tx, checkID int64, lscope scope.LabelScope) error {
	lscope = scope.NormalizeLabelScope(lscope)
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
