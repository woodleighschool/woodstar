SELECT
    ca,
    common_name,
    subject,
    issuer,
    key_algorithm,
    key_strength,
    key_usage,
    signing_algorithm,
    not_valid_after,
    not_valid_before,
    serial,
    sha1,
    CASE WHEN username = 'SYSTEM' THEN 'system' ELSE 'user' END AS source,
    username,
    path
FROM certificates
WHERE store = 'Personal';
