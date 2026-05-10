package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/db/sqlc"
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

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	db := &DB{pool: pool}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}

// Close releases database connections.
func (db *DB) Close() {
	if db != nil && db.pool != nil {
		db.pool.Close()
	}
}

// Ping checks whether the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.pool == nil {
		return errDatabaseClosed()
	}
	return db.pool.Ping(ctx)
}

// Queries returns generated database queries backed by this connection pool.
func (db *DB) Queries() *sqlc.Queries {
	if db == nil || db.pool == nil {
		return nil
	}
	return sqlc.New(db.pool)
}

// Pool returns the underlying pgxpool.Pool for callers that need raw access (e.g. scs pgxstore).
func (db *DB) Pool() *pgxpool.Pool {
	if db == nil {
		return nil
	}
	return db.pool
}

// WithTx runs fn inside a database transaction.
func (db *DB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	if db == nil || db.pool == nil {
		return errDatabaseClosed()
	}
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

func errDatabaseClosed() error {
	return errors.New("database is not open")
}
