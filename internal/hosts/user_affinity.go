package hosts

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

type UserAffinitySource string

const (
	UserAffinitySourceManual           UserAffinitySource = "manual"
	UserAffinitySourceOrbitProfile     UserAffinitySource = "orbit_profile"
	UserAffinitySourceSantaPrimaryUser UserAffinitySource = "santa_primary_user"
)

var UserAffinitySourceValues = []UserAffinitySource{
	UserAffinitySourceManual,
	UserAffinitySourceOrbitProfile,
	UserAffinitySourceSantaPrimaryUser,
}

func (UserAffinitySource) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(UserAffinitySourceValues...)
}

// UserAffinityStore persists host user affinity mappings.
type UserAffinityStore struct {
	q *sqlc.Queries
}

func NewUserAffinityStore(db *database.DB) *UserAffinityStore {
	return &UserAffinityStore{q: db.Queries()}
}

func (s *UserAffinityStore) Upsert(ctx context.Context, hostID int64, email string, source UserAffinitySource) error {
	if email == "" || source == "" {
		return nil
	}
	return s.q.UpsertHostUserAffinityMapping(ctx, sqlc.UpsertHostUserAffinityMappingParams{
		HostID: hostID,
		Email:  email,
		Source: sqlc.HostUserAffinitySource(source),
	})
}

func (s *UserAffinityStore) Delete(ctx context.Context, hostID int64, source UserAffinitySource) error {
	if source == "" {
		return nil
	}
	return s.q.DeleteHostUserAffinityMapping(ctx, sqlc.DeleteHostUserAffinityMappingParams{
		HostID: hostID,
		Source: sqlc.HostUserAffinitySource(source),
	})
}

func groupHostUserAffinityMappings(
	rows []sqlc.HostUserAffinityMapping,
	capacity int,
) map[int64][]HostUserAffinityMapping {
	grouped := make(map[int64][]HostUserAffinityMapping, capacity)
	for _, row := range rows {
		grouped[row.HostID] = append(grouped[row.HostID], hostUserAffinityMappingFromSQLC(row))
	}
	return grouped
}

func hostUserAffinityMappingFromSQLC(s sqlc.HostUserAffinityMapping) HostUserAffinityMapping {
	return HostUserAffinityMapping{
		ID:        s.ID,
		HostID:    s.HostID,
		Email:     s.Email,
		Source:    UserAffinitySource(s.Source),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
