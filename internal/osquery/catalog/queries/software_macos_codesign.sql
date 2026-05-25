SELECT
  c.path,
  c.team_identifier,
  c.cdhash_sha256
FROM apps a
JOIN codesign c ON a.path = c.path;
