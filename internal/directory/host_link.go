package directory

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

func reconcileLinks(ctx context.Context, q *sqlc.Queries) error {
	return q.ReconcileHostDirectoryLinks(ctx, sqlc.ReconcileHostDirectoryLinksParams{
		AffinitySources: []string{
			string(hosts.UserAffinitySourceOrbitProfile),
			string(hosts.UserAffinitySourceSantaPrimaryUser),
		},
	})
}
