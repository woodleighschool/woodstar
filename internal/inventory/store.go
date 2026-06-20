package inventory

import (
	"github.com/woodleighschool/woodstar/internal/database"
)

// Store persists global software titles and host inventory joins.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}
