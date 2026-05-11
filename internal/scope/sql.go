package scope

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadScope reads label scope rows for the given owner from table.
func LoadScope(ctx context.Context, pool *pgxpool.Pool, table string, ownerColumn string, ownerID int64) (LabelScope, error) {
	rows, err := pool.Query(ctx,
		fmt.Sprintf("SELECT label_id, exclude, require_all FROM %s WHERE %s = $1 ORDER BY label_id", table, ownerColumn),
		ownerID,
	)
	if err != nil {
		return LabelScope{}, err
	}
	defer rows.Close()
	return ScanScopeRows(rows)
}

// ScanScopeRows builds a LabelScope from open scope rows.
func ScanScopeRows(rows pgx.Rows) (LabelScope, error) {
	s := LabelScope{Mode: ScopeNone}
	for rows.Next() {
		var labelID int64
		var exclude bool
		var requireAll bool
		if err := rows.Scan(&labelID, &exclude, &requireAll); err != nil {
			return LabelScope{}, err
		}
		s.LabelIDs = append(s.LabelIDs, labelID)
		switch {
		case exclude:
			s.Mode = ScopeExcludeAny
		case requireAll && s.Mode != ScopeExcludeAny:
			s.Mode = ScopeIncludeAll
		case s.Mode == ScopeNone:
			s.Mode = ScopeIncludeAny
		}
	}
	if err := rows.Err(); err != nil {
		return LabelScope{}, err
	}
	return NormalizeLabelScope(s), nil
}

// ReplaceScope replaces all scope rows for the given owner inside tx.
func ReplaceScope(ctx context.Context, tx pgx.Tx, table string, ownerColumn string, ownerID int64, s LabelScope) error {
	s = NormalizeLabelScope(s)
	if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", table, ownerColumn), ownerID); err != nil {
		return err
	}
	exclude := s.Mode == ScopeExcludeAny
	requireAll := s.Mode == ScopeIncludeAll
	for _, labelID := range s.LabelIDs {
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
