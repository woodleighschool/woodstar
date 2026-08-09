package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryUnlockTimeout = 5 * time.Second

// SessionLocker holds a PostgreSQL advisory lock on one dedicated connection
// for the full duration of a callback.
type SessionLocker struct {
	pool *pgxpool.Pool
	key  int64
}

func NewSessionLocker(pool *pgxpool.Pool, key int64) *SessionLocker {
	return &SessionLocker{pool: pool, key: key}
}

// Try runs work only when this replica acquires the lock without waiting.
func (l *SessionLocker) Try(ctx context.Context, work func(context.Context) error) (bool, error) {
	conn, err := pgx.ConnectConfig(ctx, l.pool.Config().ConnConfig.Copy())
	if err != nil {
		return false, fmt.Errorf("acquire advisory-lock connection: %w", err)
	}
	closed := false
	closeConn := func() error {
		closed = true
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), advisoryUnlockTimeout)
		defer cancel()
		if err := conn.Close(closeCtx); err != nil {
			return fmt.Errorf("close advisory-lock connection: %w", err)
		}
		return nil
	}
	defer func() {
		if !closed {
			_ = closeConn()
		}
	}()

	var acquired bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", l.key).Scan(&acquired); err != nil {
		return false, errors.Join(fmt.Errorf("acquire advisory lock: %w", err), closeConn())
	}
	if !acquired {
		return false, closeConn()
	}

	workErr := work(ctx)
	unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), advisoryUnlockTimeout)
	defer cancel()
	var unlocked bool
	unlockErr := conn.QueryRow(unlockCtx, "SELECT pg_advisory_unlock($1)", l.key).Scan(&unlocked)
	if unlockErr != nil {
		unlockErr = fmt.Errorf("release advisory lock: %w", unlockErr)
	} else if !unlocked {
		unlockErr = errors.New("release advisory lock: lock was no longer held")
	}

	return true, errors.Join(workErr, unlockErr, closeConn())
}
