package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"
	"github.com/woodleighschool/goodies/pglock"
)

const (
	// migrationLockID identifies the session-level PostgreSQL advisory lock.
	migrationLockID int64 = 7146808627076917000

	// riverMigrationLockID serializes River schema migrations between replicas.
	riverMigrationLockID int64 = 7146808627076917001

	// riverMigrationVersion is deliberately pinned so dependency updates do not
	// mutate the database schema without an explicit application change.
	riverMigrationVersion = 7

	// blobyAdoptionVersion hands the existing object schema to Bloby before
	// its runner or later application migrations can use it.
	blobyAdoptionVersion = 20260901080000
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	locker, err := lock.NewPostgresSessionLocker(lock.WithLockID(migrationLockID))
	if err != nil {
		return fmt.Errorf("create migration lock: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		sqlDB,
		migrations,
		goose.WithLogger(goose.NopLogger()),
		goose.WithSessionLocker(locker),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.UpTo(ctx, blobyAdoptionVersion); err != nil {
		return fmt.Errorf("apply migrations through Bloby adoption: %w", err)
	}
	if err := blobydb.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("apply blob migrations: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if err := migrateRiver(ctx, pool); err != nil {
		return fmt.Errorf("apply River migrations: %w", err)
	}
	return nil
}

func migrateRiver(ctx context.Context, pool *pgxpool.Pool) error {
	migrator, err := rivermigrate.New(
		riverpgxv5.New(pool),
		&rivermigrate.Config{Logger: slog.New(slog.DiscardHandler)},
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	return pglock.New(pool, riverMigrationLockID).With(ctx, func(ctx context.Context) error {
		if _, err := migrator.Migrate(
			ctx,
			rivermigrate.DirectionUp,
			&rivermigrate.MigrateOpts{TargetVersion: riverMigrationVersion},
		); err != nil {
			return fmt.Errorf("migrate to version %d: %w", riverMigrationVersion, err)
		}
		return nil
	})
}
