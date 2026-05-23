SELECT
    ia.address AS primary_ip,
    id.mac AS primary_mac
FROM interface_addresses ia
JOIN interface_details id ON id.interface = ia.interface
JOIN routes r ON r.interface = ia.address
WHERE
    (r.destination = '0.0.0.0' OR r.destination = '::')
    AND r.netmask = 0
    AND r.type = 'remote'
    AND ia.address <> ''
    AND ia.address NOT LIKE '127.%'
    AND ia.address <> '::1'
ORDER BY
    r.metric ASC,
    inet_aton(ia.address) IS NOT NULL DESC,
    ia.interface
LIMIT 1;
