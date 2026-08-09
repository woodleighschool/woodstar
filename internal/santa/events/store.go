package events

import "github.com/jackc/pgx/v5/pgxpool"

// Store persists Santa execution and file-access events.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
