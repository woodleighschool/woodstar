package dbutil

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation centralizes Postgres error-code matching for stores that
// expose unique-constraint failures as ErrAlreadyExists.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsInvalidInputViolation matches database constraint/cast failures that
// should be exposed as caller input errors by stores.
func IsInvalidInputViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case "22P02", // invalid_text_representation, including enum casts
		"23502", // not_null_violation
		"23503", // foreign_key_violation
		"23514": // check_violation
		return true
	default:
		return false
	}
}
