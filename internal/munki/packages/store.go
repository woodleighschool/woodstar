package packages

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type Store struct {
	pool    *pgxpool.Pool
	objects *storage.ObjectStore
}

func NewStore(pool *pgxpool.Pool, objects *storage.ObjectStore) *Store {
	return &Store{
		pool:    pool,
		objects: objects,
	}
}
