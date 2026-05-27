package checks

import (
	"context"

	"github.com/jackc/pgx/v5"

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

	rows, err := s.db.Pool().Query(ctx,
		`SELECT id, label_scope_mode FROM checks WHERE id = ANY($1::bigint[])`, checkIDs)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var checkID int64
		var mode scope.LabelScopeMode
		if err := rows.Scan(&checkID, &mode); err != nil {
			rows.Close()
			return nil, err
		}
		scopes[checkID] = scope.LabelScope{Mode: mode}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = s.db.Pool().Query(ctx,
		`SELECT check_id, label_id FROM check_labels WHERE check_id = ANY($1::bigint[]) ORDER BY check_id, label_id`,
		checkIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var checkID int64
		var labelID int64
		if err := rows.Scan(&checkID, &labelID); err != nil {
			return nil, err
		}
		lscope := scopes[checkID]
		lscope.LabelIDs = append(lscope.LabelIDs, labelID)
		scopes[checkID] = lscope
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for checkID, lscope := range scopes {
		scopes[checkID] = scope.NormalizeLabelScope(lscope)
	}
	return scopes, nil
}

func replaceCheckScope(ctx context.Context, tx pgx.Tx, checkID int64, lscope scope.LabelScope) error {
	lscope = scope.NormalizeLabelScope(lscope)
	if _, err := tx.Exec(ctx, `UPDATE checks SET label_scope_mode = $2 WHERE id = $1`,
		checkID, string(lscope.Mode)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM check_labels WHERE check_id = $1`, checkID); err != nil {
		return err
	}
	if len(lscope.LabelIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx,
		`INSERT INTO check_labels (check_id, label_id) SELECT $1, unnest($2::bigint[])`,
		checkID, lscope.LabelIDs)
	return err
}
