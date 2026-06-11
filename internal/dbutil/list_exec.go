package dbutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListWithCount runs a paginated ListQuery: it executes the count query, then
// the page query, decoding each row into T by column name. It is the shared
// mechanics behind store List methods that map a row directly into a struct.
func ListWithCount[T any](ctx context.Context, pool *pgxpool.Pool, q ListQuery) ([]T, int, error) {
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
	defer rows.Close()
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	return items, count, err
}
