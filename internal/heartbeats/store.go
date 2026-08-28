package heartbeats

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodstar/internal/fault"
)

// Store persists the most recent heartbeat for each host and source.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Record(ctx context.Context, hostID int64, source Source, contact Contact) error {
	return record(ctx, s.pool, hostID, source, contact)
}

// RecordTx persists a heartbeat as part of an existing transaction.
func RecordTx(ctx context.Context, tx pgx.Tx, hostID int64, source Source, contact Contact) error {
	return record(ctx, tx, hostID, source, contact)
}

type executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func record(ctx context.Context, db executor, hostID int64, source Source, contact Contact) error {
	if hostID <= 0 {
		return fmt.Errorf("host ID: %w", fault.ErrInvalidInput)
	}
	if !source.valid() {
		return fmt.Errorf("source: %w", fault.ErrInvalidInput)
	}
	if contact.RemoteIP != "" {
		remoteIP, err := netip.ParseAddr(contact.RemoteIP)
		if err != nil || remoteIP.Zone() != "" {
			return fmt.Errorf("remote IP %q: %w", contact.RemoteIP, fault.ErrInvalidInput)
		}
	}

	_, err := db.Exec(ctx, `
		WITH current AS MATERIALIZED (
			SELECT true
			FROM host_heartbeats
			WHERE host_id = $1 AND source = $2
		), refreshed AS (
			UPDATE host_heartbeats
			SET
				last_seen_at = now(),
				remote_ip = NULLIF($3, '')::inet,
				user_agent = $4
			WHERE host_id = $1
			  AND source = $2
			  AND (
				last_seen_at < now() - interval '30 seconds'
				OR remote_ip IS DISTINCT FROM NULLIF($3, '')::inet
				OR user_agent IS DISTINCT FROM $4
			  )
			RETURNING true
		)
		INSERT INTO host_heartbeats (host_id, source, remote_ip, user_agent)
		SELECT $1, $2, NULLIF($3, '')::inet, $4
		WHERE NOT EXISTS (SELECT FROM current)
		ON CONFLICT (host_id, source) DO NOTHING`,
		hostID,
		source,
		contact.RemoteIP,
		contact.UserAgent,
	)
	return err
}
