-- ReconcileHostDirectoryLinks joins reported host user affinity mappings to
-- directory_users by UPN and inserts reported_user_affinity links. Existing
-- manual links are preserved by the WHERE clause; existing reported links
-- update if the preferred matched directory user has changed.
-- name: ReconcileHostDirectoryLinks :exec
INSERT INTO host_directory_user (host_id, directory_user_id, source)
SELECT host_id, directory_user_id, 'reported_user_affinity'::host_directory_user_source
FROM (
    SELECT DISTINCT ON (he.host_id)
        he.host_id,
        du.id AS directory_user_id
    FROM host_user_affinity_mappings he
    INNER JOIN directory_users du ON lower(du.user_principal_name) = lower(he.email)
    WHERE he.source::text = ANY(@affinity_sources::text[])
    ORDER BY he.host_id, CASE he.source
        WHEN 'orbit_profile' THEN 0
        WHEN 'santa_primary_user' THEN 0
        ELSE 10
    END, he.source
) preferred
ON CONFLICT (host_id) DO UPDATE SET
    directory_user_id = EXCLUDED.directory_user_id,
    source = 'reported_user_affinity',
    updated_at = now()
WHERE host_directory_user.source <> 'manual';
