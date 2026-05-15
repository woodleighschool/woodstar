SELECT
    ipv4 AS primary_ip,
    mac AS primary_mac
FROM network_interfaces
LIMIT 1;
