package events

import "github.com/woodleighschool/woodstar/internal/database"

// Store persists Santa execution and file-access events.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}
