package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

const (
	// migrationLockID identifies the session-level PostgreSQL advisory lock.
	migrationLockID int64 = 7146808627076917000

	// riverMigrationLockID serializes River schema migrations between replicas.
	riverMigrationLockID int64 = 7146808627076917001

	// riverMigrationVersion is deliberately pinned so dependency updates do not
	// mutate the database schema without an explicit application change.
	riverMigrationVersion = 7
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
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if err := migrateRiver(ctx, pool); err != nil {
		return fmt.Errorf("apply River migrations: %w", err)
	}
	return nil
}

func migrateRiver(ctx context.Context, pool *pgxpool.Pool) (err error) {
	migrator, err := rivermigrate.New(
		riverpgxv5.New(pool),
		&rivermigrate.Config{Logger: slog.New(slog.DiscardHandler)},
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	conn, err := pgx.ConnectConfig(ctx, pool.Config().ConnConfig.Copy())
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", riverMigrationLockID); err != nil {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), advisoryUnlockTimeout)
		defer cancel()
		closeErr := conn.Close(closeCtx)
		if closeErr != nil {
			closeErr = fmt.Errorf("close migration connection: %w", closeErr)
		}
		return errors.Join(fmt.Errorf("acquire migration lock: %w", err), closeErr)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), advisoryUnlockTimeout)
		defer cancel()

		var unlocked bool
		unlockErr := conn.QueryRow(
			unlockCtx,
			"SELECT pg_advisory_unlock($1)",
			riverMigrationLockID,
		).Scan(&unlocked)
		if unlockErr == nil && !unlocked {
			unlockErr = errors.New("lock was not held")
		}
		if unlockErr != nil {
			err = errors.Join(err, fmt.Errorf("release migration lock: %w", unlockErr))
		}

		closeCtx, closeCancel := context.WithTimeout(context.WithoutCancel(ctx), advisoryUnlockTimeout)
		defer closeCancel()
		if closeErr := conn.Close(closeCtx); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration connection: %w", closeErr))
		}
	}()

	if _, err := migrator.Migrate(
		ctx,
		rivermigrate.DirectionUp,
		&rivermigrate.MigrateOpts{TargetVersion: riverMigrationVersion},
	); err != nil {
		return fmt.Errorf("migrate to version %d: %w", riverMigrationVersion, err)
	}
	return nil
}
