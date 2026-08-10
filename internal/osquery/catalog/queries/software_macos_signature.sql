SELECT
  s.path,
  s.signed,
  s.identifier,
  s.cdhash,
  s.team_identifier,
  s.authority AS signing_authority
FROM apps a
JOIN signature s
  ON s.path = a.path
  AND s.hash_resources = 1
  AND s.hash_executable = 1
WHERE s.arch = '';
