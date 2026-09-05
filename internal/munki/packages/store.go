package packages

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/bloby"
)

type Store struct {
	pool    *pgxpool.Pool
	objects *bloby.Service
}

func NewStore(pool *pgxpool.Pool, objects *bloby.Service) *Store {
	return &Store{
		pool:    pool,
		objects: objects,
	}
}
