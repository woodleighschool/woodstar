package dbutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Queryer is the minimal interface satisfied by both *pgxpool.Pool and pgx.Tx.
// Use it when a helper must work inside or outside a transaction.
type Queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
