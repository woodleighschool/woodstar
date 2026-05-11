package scope

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HostMatches reports whether a host satisfies a label scope.
func HostMatches(ctx context.Context, pool *pgxpool.Pool, s LabelScope, hostID int64) (bool, error) {
	s = NormalizeLabelScope(s)
	if s.Mode == ScopeNone {
		return true, nil
	}

	var count int
	err := pool.QueryRow(
		ctx,
		`SELECT count(*)
		 FROM label_membership
		 WHERE host_id = $1 AND label_id = ANY($2)`,
		hostID,
		s.LabelIDs,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	switch s.Mode {
	case ScopeIncludeAny:
		return count > 0, nil
	case ScopeIncludeAll:
		return count == len(s.LabelIDs), nil
	case ScopeExcludeAny:
		return count == 0, nil
	default:
		return false, fmt.Errorf("scope: unknown label scope mode: %q", s.Mode)
	}
}
