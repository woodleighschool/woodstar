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
    'system' AS source,
    '' AS username,
    path
FROM certificates
WHERE path = '/Library/Keychains/System.keychain'
UNION ALL
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
    'user' AS source,
    '' AS username,
    path
FROM certificates
WHERE path LIKE '/Users/%/Library/Keychains/login.keychain-db';
