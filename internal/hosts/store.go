package hosts

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/labels"
)

// Store persists hosts.
type Store struct {
	pool   *pgxpool.Pool
	labels hostLabelReader
}

type hostLabelReader interface {
	ListForHost(ctx context.Context, hostID int64) ([]labels.Label, error)
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, labels: labels.NewStore(pool)}
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM hosts WHERE id = $1`, id)
	if err != nil {
		return dbutil.GetError(err)
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

// DeleteMany removes hosts. Missing IDs are fine.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.pool.Query(ctx, `DELETE FROM hosts WHERE id = ANY($1::bigint[]) RETURNING id`, ids)
	if err != nil {
		return 0, err
	}
	deleted, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	return len(deleted), nil
}
