SELECT
    name,
    installed,
    installed_version,
    end_time
FROM munki_installs
WHERE name <> '';
