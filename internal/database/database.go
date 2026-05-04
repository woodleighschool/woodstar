package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

// Query runs sql and returns rows from the underlying pool.
func (db *DB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if db == nil || db.pool == nil {
		return nil, errDatabaseClosed()
	}
	return db.pool.Query(ctx, sql, args...)
}

// QueryRow runs sql and returns a single row from the underlying pool.
func (db *DB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if db == nil || db.pool == nil {
		return errorRow{err: errDatabaseClosed()}
	}
	return db.pool.QueryRow(ctx, sql, args...)
}

// Exec runs sql without returning rows.
func (db *DB) Exec(ctx context.Context, sql string, args ...any) error {
	if db == nil || db.pool == nil {
		return errDatabaseClosed()
	}
	_, err := db.pool.Exec(ctx, sql, args...)
	return err
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

type errorRow struct {
	err error
}

func (r errorRow) Scan(...any) error {
	return r.err
}

func errDatabaseClosed() error {
	return errors.New("database is not open")
}
