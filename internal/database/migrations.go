package database

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5"
)

const migrationAdvisoryLockID int64 = 7146808627076917000

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (db *DB) migrate(ctx context.Context) error {
	if db == nil || db.pool == nil {
		return errDatabaseClosed()
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", migrationAdvisoryLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}

	files, err := migrationFiles()
	if err != nil {
		return err
	}
	if err := ensureMigrationsTable(ctx, tx); err != nil {
		return err
	}
	if err := applyMigrations(ctx, tx, files); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

func ensureMigrationsTable(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS migrations (
    id BIGSERIAL PRIMARY KEY,
    filename TEXT NOT NULL UNIQUE,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

func applyMigrations(ctx context.Context, tx pgx.Tx, files []string) error {
	for _, filename := range files {
		applied, err := migrationApplied(ctx, tx, filename)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := applyMigration(ctx, tx, filename); err != nil {
			return err
		}
	}
	return nil
}

func migrationApplied(ctx context.Context, tx pgx.Tx, filename string) (bool, error) {
	var count int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM migrations WHERE filename = $1", filename).
		Scan(&count); err != nil {
		return false, fmt.Errorf("check migration %s: %w", filename, err)
	}
	return count > 0, nil
}

func applyMigration(ctx context.Context, tx pgx.Tx, filename string) error {
	content, err := migrationsFS.ReadFile("migrations/" + filename)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", filename, err)
	}
	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return fmt.Errorf("execute migration %s: %w", filename, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO migrations (filename) VALUES ($1)", filename); err != nil {
		return fmt.Errorf("record migration %s: %w", filename, err)
	}
	return nil
}

func migrationFiles() ([]string, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}
