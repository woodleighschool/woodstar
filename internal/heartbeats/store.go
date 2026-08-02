package heartbeats

import (
	"context"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists the most recent heartbeat for each host and source.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Record(ctx context.Context, hostID int64, source Source, contact Contact) error {
	if hostID <= 0 {
		return fmt.Errorf("host ID: %w", dbutil.ErrInvalidInput)
	}
	if !source.valid() {
		return fmt.Errorf("source: %w", dbutil.ErrInvalidInput)
	}

	_, err := s.db.Pool().Exec(ctx, `
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
