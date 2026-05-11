package scope

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadQueryScope reads the label scope for a saved query.
func LoadQueryScope(ctx context.Context, pool *pgxpool.Pool, queryID int64) (LabelScope, error) {
	var mode LabelScopeMode
	if err := pool.QueryRow(ctx,
		"SELECT label_scope_mode FROM queries WHERE id = $1",
		queryID,
	).Scan(&mode); err != nil {
		return LabelScope{}, err
	}
	rows, err := pool.Query(ctx,
		"SELECT label_id FROM query_labels WHERE query_id = $1 ORDER BY label_id",
		queryID,
	)
	if err != nil {
		return LabelScope{}, err
	}
	defer rows.Close()
	return scanScopeRows(mode, rows)
}

// LoadCheckScope reads the label scope for a check.
func LoadCheckScope(ctx context.Context, pool *pgxpool.Pool, checkID int64) (LabelScope, error) {
	var mode LabelScopeMode
	if err := pool.QueryRow(ctx,
		"SELECT label_scope_mode FROM checks WHERE id = $1",
		checkID,
	).Scan(&mode); err != nil {
		return LabelScope{}, err
	}
	rows, err := pool.Query(ctx,
		"SELECT label_id FROM check_labels WHERE check_id = $1 ORDER BY label_id",
		checkID,
	)
	if err != nil {
		return LabelScope{}, err
	}
	defer rows.Close()
	return scanScopeRows(mode, rows)
}

// ReplaceQueryScope replaces the label scope for a saved query inside tx.
func ReplaceQueryScope(ctx context.Context, tx pgx.Tx, queryID int64, lscope LabelScope) error {
	lscope = NormalizeLabelScope(lscope)
	if _, err := tx.Exec(ctx,
		"UPDATE queries SET label_scope_mode = $2 WHERE id = $1",
		queryID, lscope.Mode,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		"DELETE FROM query_labels WHERE query_id = $1",
		queryID,
	); err != nil {
		return err
	}
	for _, labelID := range lscope.LabelIDs {
		if _, err := tx.Exec(ctx,
			"INSERT INTO query_labels (query_id, label_id) VALUES ($1, $2)",
			queryID, labelID,
		); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceCheckScope replaces the label scope for a check inside tx.
func ReplaceCheckScope(ctx context.Context, tx pgx.Tx, checkID int64, lscope LabelScope) error {
	lscope = NormalizeLabelScope(lscope)
	if _, err := tx.Exec(ctx,
		"UPDATE checks SET label_scope_mode = $2 WHERE id = $1",
		checkID, lscope.Mode,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		"DELETE FROM check_labels WHERE check_id = $1",
		checkID,
	); err != nil {
		return err
	}
	for _, labelID := range lscope.LabelIDs {
		if _, err := tx.Exec(ctx,
			"INSERT INTO check_labels (check_id, label_id) VALUES ($1, $2)",
			checkID, labelID,
		); err != nil {
			return err
		}
	}
	return nil
}

func scanScopeRows(mode LabelScopeMode, rows pgx.Rows) (LabelScope, error) {
	s := LabelScope{Mode: mode}
	for rows.Next() {
		var labelID int64
		if err := rows.Scan(&labelID); err != nil {
			return LabelScope{}, err
		}
		s.LabelIDs = append(s.LabelIDs, labelID)
	}
	if err := rows.Err(); err != nil {
		return LabelScope{}, err
	}
	return NormalizeLabelScope(s), nil
}
