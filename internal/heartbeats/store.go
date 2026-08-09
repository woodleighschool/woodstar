package heartbeats

import (
	"context"
	"fmt"
	"net/netip"

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

	_, err := s.pool.Exec(ctx, `
		INSERT INTO host_heartbeats (host_id, source, remote_ip, user_agent)
		VALUES ($1, $2, NULLIF($3, '')::inet, $4)
		ON CONFLICT (host_id, source) DO UPDATE
		SET
			last_seen_at = now(),
			remote_ip = EXCLUDED.remote_ip,
			user_agent = EXCLUDED.user_agent`,
		hostID,
		source,
		contact.RemoteIP,
		contact.UserAgent,
	)
	return err
}
