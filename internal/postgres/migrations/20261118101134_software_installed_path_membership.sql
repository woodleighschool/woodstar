-- +goose Up

DELETE FROM host_software_installed_paths path
WHERE NOT EXISTS (
    SELECT FROM host_software membership
    WHERE (membership.host_id, membership.software_id) = (path.host_id, path.software_id)
);

ALTER TABLE host_software_installed_paths
    DROP CONSTRAINT host_software_installed_paths_host_id_fkey,
    DROP CONSTRAINT host_software_installed_paths_software_id_fkey,
    ADD FOREIGN KEY (host_id, software_id) REFERENCES host_software (host_id, software_id) ON DELETE CASCADE;
