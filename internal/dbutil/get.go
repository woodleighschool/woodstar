package dbutil

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// GetOne scans exactly one row into T by column name, or ErrNotFound if no rows exist.
// q may be a *pgxpool.Pool or a pgx.Tx.
func GetOne[T any](ctx context.Context, q Queryer, sql string, args ...any) (T, error) {
	var zero T
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return zero, err
	}
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[T])
	if err != nil {
		return zero, GetError(err)
	}
	return row, nil
}

// GetAll scans every row into T by column name.
// q may be a *pgxpool.Pool or a pgx.Tx.
func GetAll[T any](ctx context.Context, q Queryer, sql string, args ...any) ([]T, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}
