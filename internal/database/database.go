package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// DB wraps the Postgres connection pool used by stores.
type DB struct {
	pool *pgxpool.Pool
}

// Open connects to Postgres and runs pending migrations.
func Open(ctx context.Context, databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := openPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	db := &DB{pool: pool}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	pool.Close()
	cfg.AfterConnect = registerConnTypes
	pool, err = openPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &DB{pool: pool}, nil
}

// Close releases database connections.
func (db *DB) Close() {
	if db != nil && db.pool != nil {
		db.pool.Close()
	}
}

// Ping checks whether the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Queries returns generated database queries backed by this connection pool.
func (db *DB) Queries() *sqlc.Queries {
	return sqlc.New(db.pool)
}

// Pool returns the underlying pgxpool.Pool for callers that need raw access (e.g. scs pgxstore).
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// WithTx runs fn inside a database transaction.
func (db *DB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func openPool(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func registerConnTypes(_ context.Context, _ *pgx.Conn) error {
	return nil
}
