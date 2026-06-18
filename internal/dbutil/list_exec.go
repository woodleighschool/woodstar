package dbutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QueryListWithCount runs a paginated ListQuery count and page query. Callers
// that need custom row mapping or enrichment own closing the returned rows.
func QueryListWithCount(ctx context.Context, pool *pgxpool.Pool, q ListQuery) (pgx.Rows, int, error) {
	countSQL, countArgs := q.BuildCount()
	var count int
	if err := pool.QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := q.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	return rows, count, nil
}

// ScanListWithCount runs a ListQuery and decodes each row with scan.
func ScanListWithCount[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	q ListQuery,
	scan func(pgx.Row) (T, error),
) ([]T, int, error) {
	rows, count, err := QueryListWithCount(ctx, pool, q)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []T{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

// ListWithCount runs a ListQuery and decodes each row into T by column name.
func ListWithCount[T any](ctx context.Context, pool *pgxpool.Pool, q ListQuery) ([]T, int, error) {
	rows, count, err := QueryListWithCount(ctx, pool, q)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	return items, count, err
}
