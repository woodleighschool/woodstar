package dbutil

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ReplaceChildren replaces a parent's ordered child rows within an existing tx:
// runs deleteSQL(deleteArgs...), then inserts each row via insertSQL using
// @named params matching R's db tags. Callers set each row's position before calling.
func ReplaceChildren[R any](
	ctx context.Context,
	q Queryer,
	deleteSQL string,
	deleteArgs []any,
	insertSQL string,
	rows []R,
) error {
	if _, err := q.Exec(ctx, deleteSQL, deleteArgs...); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := q.Exec(ctx, insertSQL, pgx.StructArgs(row)); err != nil {
			return err
		}
	}
	return nil
}
