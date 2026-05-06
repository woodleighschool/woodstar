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
  COALESCE(NULLIF(display_name, ''), NULLIF(bundle_name, ''), NULLIF(NULLIF(bundle_executable, ''), 'run.sh'), CASE WHEN name IS NOT NULL AND lower(name) LIKE '%.app' THEN substr(name, 1, length(name) - 4) ELSE name END) AS name,
  COALESCE(NULLIF(bundle_short_version, ''), bundle_version) AS version,
  bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'apps' AS source,
  '' AS vendor,
  last_opened_time AS last_opened_at,
  path AS installed_path
FROM apps
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  identifier AS extension_id,
  browser_type AS extension_for,
  'chrome_extensions' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  identifier AS extension_id,
  'firefox' AS extension_for,
  'firefox_addons' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  identifier AS extension_id,
  '' AS extension_for,
  'safari_extensions' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN safari_extensions USING (uid)
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'homebrew_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM homebrew_packages
WHERE type = 'formula'
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'homebrew_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM homebrew_packages
WHERE type = 'cask'
AND NOT EXISTS (
  SELECT 1
  FROM file
  WHERE file.path LIKE homebrew_packages.path || '/%'
  AND file.path LIKE '%.app%'
  LIMIT 1
)
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'npm_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM npm_packages;
