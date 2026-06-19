package dbutil

import (
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return ""
	}
	return pgErr.Code
}

// GetError maps missing read rows to the shared not-found sentinel.
func GetError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// MutationError maps a Postgres write error to a shared store sentinel: missing
// rows and foreign-key violations become ErrNotFound, unique violations
// ErrAlreadyExists, and value or constraint violations ErrInvalidInput.
func MutationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	switch sqlState(err) {
	case pgerrcode.ForeignKeyViolation:
		return ErrNotFound
	case pgerrcode.UniqueViolation:
		return ErrAlreadyExists
	case pgerrcode.InvalidTextRepresentation,
		pgerrcode.NotNullViolation,
		pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}
	return err
}

// DeleteConflict maps foreign-key and restrict violations to ErrConflict carrying
// message, falling back to MutationError for any other error.
func DeleteConflict(err error, message string) error {
	switch sqlState(err) {
	case pgerrcode.ForeignKeyViolation, pgerrcode.RestrictViolation:
		return fmt.Errorf("%w: %s", ErrConflict, message)
	}
	return MutationError(err)
}
