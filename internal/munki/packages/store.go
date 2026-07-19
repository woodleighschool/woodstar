package packages

import (
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type Store struct {
	db      *database.DB
	objects *storage.ObjectStore
}

func NewStore(db *database.DB, objects *storage.ObjectStore) *Store {
	return &Store{
		db:      db,
		objects: objects,
	}
}
