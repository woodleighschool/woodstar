package dbutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetOne scans exactly one row into T by column name, or ErrNotFound if no rows exist.
func GetOne[T any](ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) (T, error) {
	var zero T
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return zero, err
	}
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[T])
	if err != nil {
		return zero, GetError(err)
	}
	return row, nil
}
