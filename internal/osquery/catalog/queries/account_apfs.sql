SELECT
    c.crypto_user_uuid AS uuid,
    c.type,
    c.volume_owner
FROM apfs_volumes v
JOIN mounts m ON m.device = '/dev/' || v.device_identifier
    AND m.path = '/System/Volumes/Data'
JOIN apfs_crypto_users c ON c.device_identifier = v.device_identifier
WHERE v.role = 'Data';
