package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLState returns the PostgreSQL SQLSTATE code carried by err.
func SQLState(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return ""
	}
	return pgErr.Code
}
