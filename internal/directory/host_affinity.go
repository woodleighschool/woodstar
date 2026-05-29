package directory

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// LoadHostUserAffinity returns the preferred host/user affinity, enriched only
// when the directory reconciler has linked the host to a directory user.
func (s *Store) LoadHostUserAffinity(ctx context.Context, hostID int64) (*hosts.HostUserAffinity, error) {
	row, err := s.q.LoadHostUserAffinity(ctx, sqlc.LoadHostUserAffinityParams{AffinityHostID: hostID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil //nolint:nilnil // no host affinity is represented as an omitted object.
		}
		return nil, err
	}
	return &hosts.HostUserAffinity{
		Email:      row.Email,
		Username:   row.Username,
		Name:       row.Name,
		Department: row.Department,
		Groups:     row.Groups,
		Source:     hosts.DeviceMappingSource(row.Source),
	}, nil
}
