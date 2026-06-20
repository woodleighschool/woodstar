package directory

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const reconcileHostUserLinksSQL = `
INSERT INTO host_user_links (host_id, user_id, source)
SELECT host_id, user_id, 'reported_user_affinity'::host_user_link_source
FROM (
    SELECT DISTINCT ON (he.host_id)
        he.host_id,
        u.id AS user_id
    FROM host_user_affinity_mappings he
    INNER JOIN users u
        ON lower(u.email) = lower(he.email)
        OR lower(COALESCE(u.user_principal_name, '')) = lower(he.email)
    WHERE he.source::text = ANY($1::text[])
      AND u.deleted_at IS NULL
    ORDER BY he.host_id, CASE he.source
        WHEN 'orbit_profile' THEN 0
        WHEN 'santa_primary_user' THEN 0
        ELSE 10
    END, he.source
) preferred
ON CONFLICT (host_id) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    source = 'reported_user_affinity',
    updated_at = now()
WHERE host_user_links.source <> 'manual'`

func reconcileLinks(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, reconcileHostUserLinksSQL, []string{
		"orbit_profile",
		"santa_primary_user",
	})
	return err
}
