package db

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// migrationLockID guards concurrent migration runs across woodstar instances.
// Goose's global API does not lock; without this, a rolling restart can race
// two pods both applying the same migration.
const migrationLockID int64 = 7146808627076917000

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (db *DB) migrate(ctx context.Context) error {
	sqlDB := stdlib.OpenDBFromPool(db.pool)
	defer sqlDB.Close()

	// Pin to a single connection so the session-scoped advisory lock taken
	// below is held across every query goose issues.
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		_, _ = sqlDB.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID)
	}()

	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
