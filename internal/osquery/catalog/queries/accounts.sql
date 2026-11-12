SELECT
    u.uid,
    u.username,
    u.uuid,
    u.description AS display_name,
    u.directory AS home_directory,
    u.shell,
    u.is_hidden,
    EXISTS (
        SELECT 1 FROM logged_in_users l
        WHERE l.user = u.username AND l.tty = 'console' AND l.type = 'user'
    ) AS logged_in,
    EXISTS (
        SELECT 1
        FROM user_groups ug
        JOIN groups g ON g.gid = ug.gid
        WHERE ug.uid = u.uid AND g.groupname = 'admin'
    ) AS is_admin
FROM users u
WHERE u.username <> ''
ORDER BY u.username, u.uid;
