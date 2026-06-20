package syncstate

import (
	"github.com/woodleighschool/woodstar/internal/database"
)

// Store persists Santa sync state.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}
