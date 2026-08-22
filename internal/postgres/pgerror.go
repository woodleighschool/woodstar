package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/woodleighschool/woodstar/internal/fault"
)

func sqlState(err error) string {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return ""
	}
	return pgErr.Code
}

// GetError maps missing read rows to the shared not-found sentinel.
func GetError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fault.ErrNotFound
	}
	return err
}

// MutationError maps a Postgres write error to a shared store sentinel: missing
// rows and foreign-key violations become fault.ErrNotFound, unique violations
// fault.ErrAlreadyExists, and value or constraint violations fault.ErrInvalidInput.
func MutationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fault.ErrNotFound
	}
	switch sqlState(err) {
	case pgerrcode.ForeignKeyViolation:
		return fault.ErrNotFound
	case pgerrcode.UniqueViolation:
		return fault.ErrAlreadyExists
	case pgerrcode.InvalidTextRepresentation,
		pgerrcode.NotNullViolation,
		pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return err
}

// DeleteConflict maps foreign-key and restrict violations to fault.ErrConflict carrying
// message, falling back to MutationError for any other error.
func DeleteConflict(err error, message string) error {
	switch sqlState(err) {
	case pgerrcode.ForeignKeyViolation, pgerrcode.RestrictViolation:
		return fmt.Errorf("%w: %s", fault.ErrConflict, message)
	}
	return MutationError(err)
}
