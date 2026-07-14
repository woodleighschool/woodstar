-- +goose Up

ALTER TYPE munki_package_selection RENAME VALUE 'latest_eligible' TO 'latest';
ALTER TYPE munki_package_selection RENAME VALUE 'specific_package' TO 'specific';

ALTER TABLE munki_packages
    ALTER COLUMN installer_choices_xml DROP DEFAULT,
    ALTER COLUMN installer_choices_xml TYPE JSONB USING '[]'::JSONB,
    ALTER COLUMN installer_choices_xml SET DEFAULT '[]'::JSONB;

ALTER TABLE munki_packages
    ADD CONSTRAINT munki_packages_installer_choices_xml_check
        CHECK (jsonb_typeof(installer_choices_xml) = 'array');

UPDATE munki_packages
SET uninstall_method = CASE
        WHEN uninstall_method IN ('removepackages', 'remove_copied_items', 'uninstall_script')
            THEN uninstall_method
        ELSE ''
    END,
    restart_action = CASE
        WHEN restart_action IN ('RequireLogout', 'RecommendRestart', 'RequireRestart', 'RequireShutdown')
            THEN restart_action
        ELSE ''
    END;

ALTER TABLE munki_packages
    ALTER COLUMN uninstall_method SET DEFAULT '',
    ADD CONSTRAINT munki_packages_uninstall_method_check CHECK (
        uninstall_method IN ('', 'removepackages', 'remove_copied_items', 'uninstall_script')
    ),
    ADD CONSTRAINT munki_packages_restart_action_check CHECK (
        restart_action IN ('', 'RequireLogout', 'RecommendRestart', 'RequireRestart', 'RequireShutdown')
    ),
    ADD COLUMN blocking_applications_none BOOLEAN NOT NULL DEFAULT FALSE,
    ADD CONSTRAINT munki_packages_blocking_applications_none_check CHECK (
        NOT blocking_applications_none OR cardinality(blocking_applications) = 0
    ),
    ADD COLUMN installer_object_id BIGINT
        REFERENCES storage_objects (id) ON DELETE RESTRICT;

ALTER TABLE munki_software
    ADD COLUMN icon_object_id BIGINT
        REFERENCES storage_objects (id) ON DELETE RESTRICT;

ALTER TABLE munki_package_relations
    ADD COLUMN target_software_id BIGINT;

UPDATE munki_package_relations r
SET target_software_id = p.software_id
FROM munki_packages p
WHERE p.id = r.target_package_id;

ALTER TABLE munki_package_relations
    ALTER COLUMN target_software_id SET NOT NULL,
    ALTER COLUMN target_package_id DROP NOT NULL,
    ADD CONSTRAINT munki_package_relations_target_software_id_fkey
        FOREIGN KEY (target_software_id)
        REFERENCES munki_software (id) ON DELETE RESTRICT,
    ADD CONSTRAINT munki_package_relations_target_software_package_fkey
        FOREIGN KEY (target_software_id, target_package_id)
        REFERENCES munki_packages (software_id, id) ON DELETE RESTRICT;

DROP INDEX munki_software_icon_artifact_idx;
DROP INDEX munki_packages_installer_artifact_idx;
DROP INDEX munki_packages_uninstaller_artifact_idx;

ALTER TABLE munki_software
    DROP COLUMN icon_artifact_id,
    DROP COLUMN icon_name,
    DROP COLUMN icon_hash;

ALTER TABLE munki_packages
    DROP COLUMN installer_artifact_id,
    DROP COLUMN uninstaller_artifact_id;

DROP TABLE munki_artifacts;
DROP TYPE munki_artifact_kind;

CREATE INDEX munki_software_icon_object_idx
    ON munki_software (icon_object_id);
CREATE INDEX munki_packages_installer_object_idx
    ON munki_packages (installer_object_id);
CREATE INDEX munki_package_relations_target_software_idx
    ON munki_package_relations (target_software_id);
