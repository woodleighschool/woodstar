package queries

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

func (s *QueryStore) loadScope(
	ctx context.Context,
	table string,
	ownerColumn string,
	ownerID int64,
) (hosts.LabelScope, error) {
	rows, err := s.db.Pool().Query(ctx,
		fmt.Sprintf("SELECT label_id, exclude, require_all FROM %s WHERE %s = $1 ORDER BY label_id", table, ownerColumn),
		ownerID,
	)
	if err != nil {
		return hosts.LabelScope{}, err
	}
	defer rows.Close()
	return scanScopeRows(rows)
}

func scanScopeRows(rows pgx.Rows) (hosts.LabelScope, error) {
	scope := hosts.LabelScope{Mode: hosts.ScopeNone}
	for rows.Next() {
		var labelID int64
		var exclude bool
		var requireAll bool
		if err := rows.Scan(&labelID, &exclude, &requireAll); err != nil {
			return hosts.LabelScope{}, err
		}
		scope.LabelIDs = append(scope.LabelIDs, labelID)
		switch {
		case exclude:
			scope.Mode = hosts.ScopeExcludeAny
		case requireAll && scope.Mode != hosts.ScopeExcludeAny:
			scope.Mode = hosts.ScopeIncludeAll
		case scope.Mode == hosts.ScopeNone:
			scope.Mode = hosts.ScopeIncludeAny
		}
	}
	if err := rows.Err(); err != nil {
		return hosts.LabelScope{}, err
	}
	return hosts.NormalizeLabelScope(scope), nil
}

func replaceScope(
	ctx context.Context,
	tx pgx.Tx,
	table string,
	ownerColumn string,
	ownerID int64,
	scope hosts.LabelScope,
) error {
	scope = hosts.NormalizeLabelScope(scope)
	if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", table, ownerColumn), ownerID); err != nil {
		return err
	}
	exclude := scope.Mode == hosts.ScopeExcludeAny
	requireAll := scope.Mode == hosts.ScopeIncludeAll
	for _, labelID := range scope.LabelIDs {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf(
				"INSERT INTO %s (%s, label_id, exclude, require_all) VALUES ($1, $2, $3, $4)",
				table,
				ownerColumn,
			),
			ownerID,
			labelID,
			exclude,
			requireAll,
		); err != nil {
			return err
		}
	}
	return nil
}
