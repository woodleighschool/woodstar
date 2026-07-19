package hosts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
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
	return openapischema.StringEnum(PrimaryUserSourceValues...)
}

// PrimaryUserStore persists host primary-user source observations.
type PrimaryUserStore struct {
	db     *database.DB
	labels *labels.Store
}

func NewPrimaryUserStore(db *database.DB) *PrimaryUserStore {
	return &PrimaryUserStore{db: db, labels: labels.NewStore(db)}
}

func (s *PrimaryUserStore) Upsert(ctx context.Context, hostID int64, email string, source PrimaryUserSource) error {
	email, err := validatePrimaryUser(email, source)
	if err != nil {
		return err
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := upsertPrimaryUser(ctx, tx, hostID, email, source); err != nil {
			return err
		}
		return s.labels.RefreshDerivedTx(ctx, tx)
	})
}

func upsertPrimaryUser(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	email string,
	source PrimaryUserSource,
) error {
	_, err := tx.Exec(ctx, upsertHostPrimaryUserSourceSQL, pgx.NamedArgs{
		"host_id": hostID,
		"email":   email,
		"source":  string(source),
	})
	return dbutil.MutationError(err)
}

func (s *PrimaryUserStore) Delete(ctx context.Context, hostID int64, source PrimaryUserSource) error {
	if err := validatePrimaryUserSource(source); err != nil {
		return err
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := deletePrimaryUser(ctx, tx, hostID, source); err != nil {
			return err
		}
		return s.labels.RefreshDerivedTx(ctx, tx)
	})
}

func deletePrimaryUser(ctx context.Context, tx pgx.Tx, hostID int64, source PrimaryUserSource) error {
	tag, err := tx.Exec(ctx, deleteHostPrimaryUserSourceSQL, pgx.NamedArgs{
		"host_id": hostID,
		"source":  string(source),
	})
	if err != nil {
		return dbutil.MutationError(err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	exists, err := hostExists(ctx, tx, hostID)
	if err != nil {
		return err
	}
	if !exists {
		return dbutil.ErrNotFound
	}
	return nil
}

type primaryUserMutation struct {
	Email  string            `validate:"required,email"`
	Source PrimaryUserSource `validate:"required,oneof=manual orbit_profile"`
}

func validatePrimaryUser(email string, source PrimaryUserSource) (string, error) {
	email = strings.TrimSpace(email)
	mutation := primaryUserMutation{Email: email, Source: source}
	if err := validation.Struct(mutation); err != nil {
		return "", fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return email, nil
}

func validatePrimaryUserSource(source PrimaryUserSource) error {
	if err := validation.Struct(struct {
		Source PrimaryUserSource `validate:"required,oneof=manual orbit_profile"`
	}{Source: source}); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func hostExists(ctx context.Context, tx pgx.Tx, hostID int64) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM hosts WHERE id = $1)`, hostID).Scan(&exists)
	return exists, err
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

func (s *Store) attachPrimaryUser(ctx context.Context, hosts []Host) error {
	if len(hosts) == 0 {
		return nil
	}
	hostIDs := make([]int64, len(hosts))
	for i := range hosts {
		hostIDs[i] = hosts[i].ID
	}
	primaryUsers, err := s.loadPrimaryUser(ctx, hostIDs)
	if err != nil {
		return err
	}
	for i := range hosts {
		primaryUser := primaryUsers[hosts[i].ID]
		hosts[i].PrimaryUser = primaryUser.Primary
		hosts[i].PrimaryUserSources = primaryUser.Sources
	}
	return nil
}

func (s *Store) loadPrimaryUser(ctx context.Context, hostIDs []int64) (map[int64]hostPrimaryUser, error) {
	primaryUsers := make(map[int64]hostPrimaryUser, len(hostIDs))
	for _, hostID := range hostIDs {
		primaryUsers[hostID] = hostPrimaryUser{Sources: []HostPrimaryUserSource{}}
	}
	if len(hostIDs) == 0 {
		return primaryUsers, nil
	}

	sourceRows, err := s.db.Pool().Query(ctx, listHostPrimaryUserSourcesForHostsSQL, hostIDs)
	if err != nil {
		return nil, err
	}
	sources, err := pgx.CollectRows(sourceRows, pgx.RowToStructByName[hostPrimaryUserSourceRow])
	if err != nil {
		return nil, err
	}
	grouped := groupHostPrimaryUserSources(sources)

	primaryRows, err := s.db.Pool().Query(ctx, listHostPrimaryUsersSQL, hostIDs)
	if err != nil {
		return nil, err
	}
	primaries, err := pgx.CollectRows(primaryRows, pgx.RowToStructByName[hostPrimaryUserRow])
	if err != nil {
		return nil, err
	}
	for _, row := range primaries {
		primaryUser := primaryUsers[row.HostID]
		primaryUser.Primary = &HostPrimaryUser{
			Email:      row.Email,
			Username:   row.Username,
			Name:       row.Name,
			Department: row.Department,
			Groups:     row.Groups,
			Source:     PrimaryUserSource(row.Source),
		}
		primaryUsers[row.HostID] = primaryUser
	}
	for hostID, sources := range grouped {
		primaryUser := primaryUsers[hostID]
		primaryUser.Sources = sources
		primaryUsers[hostID] = primaryUser
	}
	return primaryUsers, nil
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
