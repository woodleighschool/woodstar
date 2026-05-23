SELECT name, value
FROM osquery_flags
WHERE name = 'distributed_interval'
UNION ALL
SELECT 'config_tls_refresh' AS name, value
FROM osquery_flags
WHERE name = 'config_tls_refresh'
UNION ALL
SELECT 'config_tls_refresh' AS name, value
FROM osquery_flags
WHERE name = 'config_refresh'
AND NOT EXISTS (
  SELECT 1
  FROM osquery_flags
  WHERE name = 'config_tls_refresh'
);
