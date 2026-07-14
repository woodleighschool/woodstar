package dbutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func queryListWithCount(ctx context.Context, pool *pgxpool.Pool, q ListQuery) (pgx.Rows, int, error) {
	query, args, err := q.Build()
	if err != nil {
		return nil, 0, err
	}
	countSQL, countArgs := q.BuildCount()
	var count int
	if err := pool.QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	return rows, count, nil
}

// ListWithCount runs a ListQuery and decodes each row into T by column name.
func ListWithCount[T any](ctx context.Context, pool *pgxpool.Pool, q ListQuery) ([]T, int, error) {
	rows, count, err := queryListWithCount(ctx, pool, q)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	return items, count, err
}
