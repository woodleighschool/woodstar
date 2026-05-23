SELECT
  name,
  version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS extension_for,
  'python_packages' AS source,
  '' AS vendor,
  '' AS arch,
  '' AS release,
  '' AS last_opened_at,
  path AS installed_path
FROM python_packages;
