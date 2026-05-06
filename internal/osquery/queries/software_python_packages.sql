WITH cached_users AS (
  SELECT uid
  FROM users
  WHERE type <> 'special'
  AND shell NOT LIKE '%/false'
  AND shell NOT LIKE '%/nologin'
  AND shell NOT LIKE '%/shutdown'
  AND shell NOT LIKE '%/halt'
)
SELECT
  name,
  version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'python_packages' AS source,
  '' AS vendor,
  '' AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN python_packages USING (uid);
