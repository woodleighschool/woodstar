package directory

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

// LoadHostUserAffinity returns the preferred host/user affinity, enriched only
// when the directory reconciler has linked the host to a directory user.
func (s *Store) LoadHostUserAffinity(ctx context.Context, hostID int64) (*hosts.HostUserAffinity, error) {
	rows, err := s.db.Pool().Query(ctx, hostUserAffinitySQL, hostID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostUserAffinityRecord])
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil //nolint:nilnil // no host affinity is represented as an omitted object.
	}
	record := records[0]
	return &hosts.HostUserAffinity{
		Email:      record.Email,
		Username:   record.Username,
		Name:       record.Name,
		Department: record.Department,
		Groups:     record.Groups,
		Source:     hosts.DeviceMappingSource(record.Source),
	}, nil
}

type hostUserAffinityRecord struct {
	Email      string
	Source     string
	Username   string
	Name       string
	Department string
	Groups     []string
}

const hostUserAffinitySQL = `
WITH primary_mapping AS (
	SELECT host_id, email, source::text AS source
	FROM host_emails
	WHERE host_id = $1
	ORDER BY CASE source
		WHEN 'manual' THEN 0
		WHEN 'orbit_profile' THEN 1
		WHEN 'santa_primary_user' THEN 1
		ELSE 10
	END, source
	LIMIT 1
)
SELECT
	pm.email,
	pm.source,
	COALESCE(du.mail_nickname, '') AS username,
	COALESCE(du.display_name, '') AS name,
	COALESCE(du.department, '') AS department,
	COALESCE(
		array_agg(dg.display_name ORDER BY lower(dg.display_name)) FILTER (WHERE dg.id IS NOT NULL),
		ARRAY[]::text[]
	) AS groups
FROM primary_mapping pm
LEFT JOIN host_directory_user hdu ON hdu.host_id = pm.host_id
LEFT JOIN directory_users du ON du.id = hdu.directory_user_id AND du.active
LEFT JOIN directory_user_groups dug ON dug.directory_user_id = du.id
LEFT JOIN directory_groups dg ON dg.id = dug.directory_group_id
GROUP BY pm.email, pm.source, du.mail_nickname, du.display_name, du.department`
