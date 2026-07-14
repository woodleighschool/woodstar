SELECT
    name,
    installed,
    installed_version
FROM munki_installs
WHERE name <> '';
