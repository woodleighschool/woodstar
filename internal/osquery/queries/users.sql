SELECT
    uid,
    username,
    type,
    description,
    directory,
    shell
FROM users
WHERE
    username <> ''
    AND type <> 'special'
    AND shell NOT LIKE '%/false'
    AND shell NOT LIKE '%/nologin'
    AND shell NOT LIKE '%/shutdown'
    AND shell NOT LIKE '%/halt'
ORDER BY username, uid;
