SELECT name, bundle_short_version AS version, 'app' AS source, bundle_identifier, path, last_opened_time FROM apps
UNION ALL
SELECT name, version, 'brew' AS source, '' AS bundle_identifier, '' AS path, NULL AS last_opened_time FROM homebrew_packages;
