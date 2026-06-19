package directory

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

func reconcileLinks(ctx context.Context, q *sqlc.Queries) error {
	return q.ReconcileHostUserLinks(ctx, sqlc.ReconcileHostUserLinksParams{
		AffinitySources: []string{
			string(sqlc.HostUserAffinitySourceOrbitProfile),
			string(sqlc.HostUserAffinitySourceSantaPrimaryUser),
		},
	})
}
