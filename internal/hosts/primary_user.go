package hosts

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

type PrimaryUserSource string

const (
	PrimaryUserSourceManual       PrimaryUserSource = "manual"
	PrimaryUserSourceOrbitProfile PrimaryUserSource = "orbit_profile"
)

var PrimaryUserSourceValues = []PrimaryUserSource{
	PrimaryUserSourceManual,
	PrimaryUserSourceOrbitProfile,
}

func (PrimaryUserSource) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(PrimaryUserSourceValues...)
}

// PrimaryUserStore persists host primary-user source observations.
type PrimaryUserStore struct {
	db *database.DB
}

func NewPrimaryUserStore(db *database.DB) *PrimaryUserStore {
	return &PrimaryUserStore{db: db}
}

func (s *PrimaryUserStore) Upsert(ctx context.Context, hostID int64, email string, source PrimaryUserSource) error {
	if email == "" || source == "" {
		return nil
	}
	_, err := s.db.Pool().Exec(ctx, upsertHostPrimaryUserSourceSQL, pgx.NamedArgs{
		"host_id": hostID,
		"email":   email,
		"source":  string(source),
	})
	return err
}

func (s *PrimaryUserStore) Delete(ctx context.Context, hostID int64, source PrimaryUserSource) error {
	if source == "" {
		return nil
	}
	_, err := s.db.Pool().Exec(ctx, deleteHostPrimaryUserSourceSQL, pgx.NamedArgs{
		"host_id": hostID,
		"source":  string(source),
	})
	return err
}

type hostPrimaryUserSourceRow struct {
	ID        int64     `db:"id"`
	HostID    int64     `db:"host_id"`
	Email     string    `db:"email"`
	Source    string    `db:"source"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func groupHostPrimaryUserSources(rows []hostPrimaryUserSourceRow) map[int64][]HostPrimaryUserSource {
	grouped := make(map[int64][]HostPrimaryUserSource)
	for _, row := range rows {
		grouped[row.HostID] = append(grouped[row.HostID], hostPrimaryUserSourceFromRow(row))
	}
	return grouped
}

func hostPrimaryUserSourceFromRow(row hostPrimaryUserSourceRow) HostPrimaryUserSource {
	return HostPrimaryUserSource{
		ID:        row.ID,
		HostID:    row.HostID,
		Email:     row.Email,
		Source:    PrimaryUserSource(row.Source),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

type hostPrimaryUserRow struct {
	HostID     int64    `db:"host_id"`
	Email      string   `db:"email"`
	Source     string   `db:"source"`
	Username   string   `db:"username"`
	Name       string   `db:"name"`
	Department string   `db:"department"`
	Groups     []string `db:"groups"`
}

type hostPrimaryUser struct {
	Primary *HostPrimaryUser
	Sources []HostPrimaryUserSource
}

const primaryUserSourceOrderSQL = `CASE source
	WHEN 'manual' THEN 0
	WHEN 'orbit_profile' THEN 1
	ELSE 10
END`

const upsertHostPrimaryUserSourceSQL = `
INSERT INTO host_primary_user_sources (
	host_id,
	email,
	source
)
VALUES (
	@host_id,
	@email,
	@source::host_primary_user_source
)
ON CONFLICT (host_id, source) DO UPDATE SET
	email = EXCLUDED.email,
	updated_at = now()`

const deleteHostPrimaryUserSourceSQL = `
DELETE FROM host_primary_user_sources
WHERE host_id = @host_id
  AND source = @source::host_primary_user_source`

const listHostPrimaryUserSourcesForHostsSQL = `
SELECT id, host_id, email, source, created_at, updated_at
FROM host_primary_user_sources
WHERE host_id = ANY($1::bigint[])
ORDER BY host_id, ` + primaryUserSourceOrderSQL + `, source`

const listHostPrimaryUsersSQL = `
WITH preferred AS (
	SELECT DISTINCT ON (host_id)
		host_id,
		email,
		source::text AS source
	FROM host_primary_user_sources
	WHERE host_id = ANY($1::bigint[])
	ORDER BY host_id, ` + primaryUserSourceOrderSQL + `, source
),
resolved AS (
	SELECT
		p.host_id,
		p.email,
		p.source,
		u.id AS user_id,
		COALESCE(u.mail_nickname, '') AS username,
		COALESCE(u.name, '') AS name,
		COALESCE(u.department, '') AS department
	FROM preferred p
	LEFT JOIN LATERAL (
		SELECT u.id, u.mail_nickname, u.name, u.department
		FROM users u
		WHERE u.deleted_at IS NULL
		  AND (
			lower(u.email) = lower(p.email)
			OR (
				u.user_principal_name IS NOT NULL
				AND lower(u.user_principal_name) = lower(p.email)
			)
		  )
		ORDER BY CASE WHEN lower(u.email) = lower(p.email) THEN 0 ELSE 1 END, u.id
		LIMIT 1
	) u ON true
)
SELECT
	r.host_id,
	r.email,
	r.source,
	r.username,
	r.name,
	r.department,
	COALESCE(
		array_agg(eg.display_name ORDER BY lower(eg.display_name)) FILTER (WHERE eg.id IS NOT NULL),
		ARRAY[]::text[]
	)::text[] AS groups
FROM resolved r
LEFT JOIN directory_group_memberships egm ON egm.user_id = r.user_id
LEFT JOIN directory_groups eg ON eg.id = egm.group_id
GROUP BY r.host_id, r.email, r.source, r.username, r.name, r.department
ORDER BY r.host_id`
