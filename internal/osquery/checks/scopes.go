package checks

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/osquery/labelscope"
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
	if len(checkIDs) == 0 {
		return map[int64]scope.LabelScope{}, nil
	}
	return labelscope.LoadChecks(ctx, s.q, checkIDs)
}

func replaceCheckScope(ctx context.Context, tx pgx.Tx, checkID int64, lscope scope.LabelScope) error {
	return labelscope.ReplaceCheck(ctx, tx, checkID, lscope)
}

func storedLabelScope(lscope scope.LabelScope) scope.LabelScope {
	return labelscope.Normalize(lscope)
}
