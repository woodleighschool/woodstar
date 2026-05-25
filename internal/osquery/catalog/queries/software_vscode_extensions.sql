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
  vscode_extensions.uuid AS extension_id,
  vscode_edition AS extension_for,
  'vscode_extensions' AS source,
  publisher AS vendor,
  '' AS arch,
  '' AS release,
  '' AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN vscode_extensions USING (uid);
