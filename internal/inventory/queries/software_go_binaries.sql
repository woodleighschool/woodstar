SELECT
  COALESCE(NULLIF(module_path, ''), name) AS name,
  version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'go_binaries' AS source,
  '' AS vendor,
  '' AS last_opened_at,
  installed_path
FROM go_binaries;
