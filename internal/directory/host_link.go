package directory

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// HostLink is the device-to-directory-user mapping.
type HostLink struct {
	HostID          int64     `json:"host_id"`
	DirectoryUserID int64     `json:"directory_user_id"`
	Source          string    `json:"source"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Host link source values.
const (
	HostLinkSourceManual   = "manual"
	HostLinkSourceMDMEmail = "mdm_email"
)

// GetHostLink returns the directory-user link for hostID, or ErrNotFound.
func (s *Store) GetHostLink(ctx context.Context, hostID int64) (*HostLink, error) {
	row, err := s.q.GetHostDirectoryUser(ctx, sqlc.GetHostDirectoryUserParams{HostID: hostID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	l := hostLinkFromSQLC(row)
	return &l, nil
}

// SetManualHostLink writes an admin-asserted link. Manual writes overwrite
// any prior source.
func (s *Store) SetManualHostLink(ctx context.Context, hostID, directoryUserID int64) (*HostLink, error) {
	row, err := s.q.ManualSetHostDirectoryUser(ctx, sqlc.ManualSetHostDirectoryUserParams{
		HostID:          hostID,
		DirectoryUserID: directoryUserID,
	})
	if err != nil {
		return nil, err
	}
	l := hostLinkFromSQLC(row)
	return &l, nil
}

// DeleteHostLink removes the directory link for hostID.
func (s *Store) DeleteHostLink(ctx context.Context, hostID int64) error {
	return s.q.DeleteHostDirectoryUser(ctx, sqlc.DeleteHostDirectoryUserParams{HostID: hostID})
}

// ReconcileLinks joins host_emails(source=orbit_profile) to directory_users
// by user_principal_name and upserts the resulting mdm_email links. Manual
// links are preserved by the SQL WHERE clause. Returns no count today; the
// caller logs around it for visibility.
func (s *Store) ReconcileLinks(ctx context.Context) error {
	return s.q.ReconcileHostDirectoryLinks(ctx, sqlc.ReconcileHostDirectoryLinksParams{
		MdmSource: hosts.DeviceMappingSourceOrbitProfile,
	})
}

func hostLinkFromSQLC(s sqlc.HostDirectoryUser) HostLink {
	return HostLink{
		HostID:          s.HostID,
		DirectoryUserID: s.DirectoryUserID,
		Source:          s.Source,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}
