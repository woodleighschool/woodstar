SELECT
    name,
    display_name,
    installed,
    installed_version,
    version_to_install
FROM munki_installs
WHERE name <> '';
