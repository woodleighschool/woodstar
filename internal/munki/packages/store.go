package packages

import (
	"log/slog"

	"github.com/woodleighschool/woodstar/internal/database"
)

type Store struct {
	db      *database.DB
	objects objectStore
	logger  *slog.Logger
}

func NewStore(db *database.DB, objects objectStore, logger *slog.Logger) *Store {
	return &Store{
		db:      db,
		objects: objects,
		logger:  logger,
	}
}
