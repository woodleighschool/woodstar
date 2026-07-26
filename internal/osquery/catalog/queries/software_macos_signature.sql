SELECT
  s.path,
  s.identifier,
  s.team_identifier,
  s.authority AS signing_authority
FROM apps a
JOIN signature s
  ON s.path = a.path
  AND s.hash_resources = 0
  AND s.hash_executable = 0
WHERE s.arch = '';
