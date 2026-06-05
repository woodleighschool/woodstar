package munki

import (
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func mapDesiredMutationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	switch database.SQLState(err) {
	case pgerrcode.ForeignKeyViolation:
		return dbutil.ErrNotFound
	case pgerrcode.UniqueViolation:
		return dbutil.ErrAlreadyExists
	case pgerrcode.InvalidTextRepresentation,
		pgerrcode.NotNullViolation,
		pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return err
}
