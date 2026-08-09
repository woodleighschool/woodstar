package syncstate

import "github.com/jackc/pgx/v5/pgxpool"

// Store persists Santa sync state.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
