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
  'programs' AS source,
  publisher AS vendor,
  '' AS arch,
  '' AS release,
  0 AS last_opened_at,
  install_location AS installed_path
FROM programs
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'ie_extensions' AS source,
  '' AS vendor,
  '' AS arch,
  '' AS release,
  0 AS last_opened_at,
  path AS installed_path
FROM ie_extensions
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  identifier AS extension_id,
  browser_type AS extension_for,
  'chrome_extensions' AS source,
  '' AS vendor,
  '' AS arch,
  '' AS release,
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
  '' AS arch,
  '' AS release,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION ALL
SELECT
  name,
  version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'chocolatey_packages' AS source,
  '' AS vendor,
  '' AS arch,
  '' AS release,
  0 AS last_opened_at,
  path AS installed_path
FROM chocolatey_packages;
