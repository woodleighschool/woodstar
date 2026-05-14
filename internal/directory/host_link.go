package directory

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// ReconcileLinks joins host_emails(source=orbit_profile) to directory_users
// by user_principal_name and upserts the resulting mdm_email links. Manual
// links are preserved by the SQL WHERE clause. Returns no count today; the
// caller logs around it for visibility.
func (s *Store) ReconcileLinks(ctx context.Context) error {
	return s.q.ReconcileHostDirectoryLinks(ctx, sqlc.ReconcileHostDirectoryLinksParams{
		MdmSource: hosts.DeviceMappingSourceOrbitProfile,
	})
}
