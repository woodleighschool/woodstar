package inventory

import "github.com/jackc/pgx/v5/pgxpool"

// Store persists global software titles and host inventory joins.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
