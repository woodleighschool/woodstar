package hosts

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
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
	db *database.DB
}

func NewUserAffinityStore(db *database.DB) *UserAffinityStore {
	return &UserAffinityStore{db: db}
}

func (s *UserAffinityStore) Upsert(ctx context.Context, hostID int64, email string, source UserAffinitySource) error {
	if email == "" || source == "" {
		return nil
	}
	_, err := s.db.Pool().Exec(ctx, upsertHostUserAffinityMappingSQL, pgx.NamedArgs{
		"host_id": hostID,
		"email":   email,
		"source":  string(source),
	})
	return err
}

func (s *UserAffinityStore) Delete(ctx context.Context, hostID int64, source UserAffinitySource) error {
	if source == "" {
		return nil
	}
	_, err := s.db.Pool().Exec(ctx, deleteHostUserAffinityMappingSQL, pgx.NamedArgs{
		"host_id": hostID,
		"source":  string(source),
	})
	return err
}

type hostUserAffinityMappingRow struct {
	ID        int64     `db:"id"`
	HostID    int64     `db:"host_id"`
	Email     string    `db:"email"`
	Source    string    `db:"source"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func groupHostUserAffinityMappings(rows []hostUserAffinityMappingRow) map[int64][]HostUserAffinityMapping {
	grouped := make(map[int64][]HostUserAffinityMapping)
	for _, row := range rows {
		grouped[row.HostID] = append(grouped[row.HostID], hostUserAffinityMappingFromRow(row))
	}
	return grouped
}

func hostUserAffinityMappingFromRow(row hostUserAffinityMappingRow) HostUserAffinityMapping {
	return HostUserAffinityMapping{
		ID:        row.ID,
		HostID:    row.HostID,
		Email:     row.Email,
		Source:    UserAffinitySource(row.Source),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

type hostUserAffinityPrimaryRow struct {
	HostID     int64    `db:"host_id"`
	Email      string   `db:"email"`
	Source     string   `db:"source"`
	Username   string   `db:"username"`
	Name       string   `db:"name"`
	Department string   `db:"department"`
	Groups     []string `db:"groups"`
}

const upsertHostUserAffinityMappingSQL = `
INSERT INTO host_user_affinity_mappings (
	host_id,
	email,
	source
)
VALUES (
	@host_id,
	@email,
	@source::host_user_affinity_source
)
ON CONFLICT (host_id, source) DO UPDATE SET
	email = EXCLUDED.email,
	updated_at = now()`

const deleteHostUserAffinityMappingSQL = `
DELETE FROM host_user_affinity_mappings
WHERE host_id = @host_id
  AND source = @source::host_user_affinity_source`

const listHostUserAffinityMappingsForHostsSQL = `
SELECT id, host_id, email, source, created_at, updated_at
FROM host_user_affinity_mappings
WHERE host_id = ANY($1::bigint[])
ORDER BY host_id, CASE source
	WHEN 'manual' THEN 0
	WHEN 'orbit_profile' THEN 1
	WHEN 'santa_primary_user' THEN 1
	ELSE 10
END, source`

const listHostUserAffinityPrimariesSQL = `
WITH primary_mapping AS (
	SELECT DISTINCT ON (he.host_id) he.host_id, he.email, he.source::text AS source
	FROM host_user_affinity_mappings he
	WHERE he.host_id = ANY($1::bigint[])
	ORDER BY he.host_id, CASE he.source
		WHEN 'manual' THEN 0
		WHEN 'orbit_profile' THEN 1
		WHEN 'santa_primary_user' THEN 1
		ELSE 10
	END, he.source
)
SELECT
	pm.host_id,
	pm.email,
	pm.source,
	COALESCE(u.mail_nickname, '') AS username,
	COALESCE(u.name, '') AS name,
	COALESCE(u.department, '') AS department,
	COALESCE(
		array_agg(eg.display_name ORDER BY lower(eg.display_name)) FILTER (WHERE eg.id IS NOT NULL),
		ARRAY[]::text[]
	)::text[] AS groups
FROM primary_mapping pm
LEFT JOIN host_user_links hul ON hul.host_id = pm.host_id
LEFT JOIN users u ON u.id = hul.user_id AND u.deleted_at IS NULL
LEFT JOIN directory_group_memberships egm ON egm.user_id = u.id
LEFT JOIN directory_groups eg ON eg.id = egm.group_id
GROUP BY pm.host_id, pm.email, pm.source, u.mail_nickname, u.name, u.department
ORDER BY pm.host_id`
