SELECT
    name,
    value
FROM osquery_flags
WHERE name IN ('distributed_interval', 'config_tls_refresh', 'config_refresh');
