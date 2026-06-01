-- +goose Up

CREATE TYPE munki_deployment_action AS ENUM (
    'install',
    'remove',
    'update_if_present',
    'none'
);

CREATE TYPE munki_self_service_mode AS ENUM (
    'hidden',
    'available',
    'featured',
    'default'
);

CREATE TYPE munki_package_selection AS ENUM (
    'latest_eligible',
    'specific_package'
);

ALTER TABLE munki_deployments
    ADD COLUMN software_id BIGINT REFERENCES munki_software_titles (id) ON DELETE CASCADE,
    ADD COLUMN action munki_deployment_action,
    ADD COLUMN self_service munki_self_service_mode,
    ADD COLUMN package_selection munki_package_selection,
    ADD COLUMN pinned_package_id BIGINT;

UPDATE munki_deployments d
SET
    software_id = p.software_id,
    action = CASE d.intent
        WHEN 'ensure_installed' THEN 'install'::munki_deployment_action
        WHEN 'ensure_absent' THEN 'remove'::munki_deployment_action
        WHEN 'update_if_present' THEN 'update_if_present'::munki_deployment_action
        ELSE 'none'::munki_deployment_action
    END,
    self_service = CASE d.intent
        WHEN 'optional' THEN 'available'::munki_self_service_mode
        WHEN 'featured' THEN 'featured'::munki_self_service_mode
        ELSE 'hidden'::munki_self_service_mode
    END,
    package_selection = 'specific_package'::munki_package_selection,
    pinned_package_id = d.package_id
FROM munki_packages p
WHERE p.id = d.package_id;

ALTER TABLE munki_deployments
    ALTER COLUMN software_id SET NOT NULL,
    ALTER COLUMN action SET NOT NULL,
    ALTER COLUMN action SET DEFAULT 'install',
    ALTER COLUMN self_service SET NOT NULL,
    ALTER COLUMN self_service SET DEFAULT 'hidden',
    ALTER COLUMN package_selection SET NOT NULL,
    ALTER COLUMN package_selection SET DEFAULT 'latest_eligible';

ALTER TABLE munki_deployments
    ADD CONSTRAINT munki_deployments_package_selection_check
    CHECK (
        (package_selection = 'latest_eligible' AND pinned_package_id IS NULL)
        OR (package_selection = 'specific_package' AND pinned_package_id IS NOT NULL)
    );

ALTER TABLE munki_packages
    ADD CONSTRAINT munki_packages_software_id_id_key UNIQUE (software_id, id);

ALTER TABLE munki_deployments
    ADD CONSTRAINT munki_deployments_pinned_package_software_fkey
    FOREIGN KEY (software_id, pinned_package_id)
    REFERENCES munki_packages (software_id, id)
    ON DELETE RESTRICT;

DROP INDEX IF EXISTS munki_deployments_package_idx;
DROP INDEX IF EXISTS munki_deployments_position_idx;

ALTER TABLE munki_deployments
    DROP COLUMN package_id,
    DROP COLUMN intent;

DROP TYPE munki_deployment_intent;

CREATE INDEX munki_deployments_software_idx
    ON munki_deployments (software_id);
CREATE INDEX munki_deployments_pinned_package_idx
    ON munki_deployments (pinned_package_id);
CREATE INDEX munki_deployments_position_idx
    ON munki_deployments (software_id, position, id);

-- +goose Down

CREATE TYPE munki_deployment_intent AS ENUM (
    'ensure_installed',
    'ensure_absent',
    'update_if_present',
    'optional',
    'featured'
);

ALTER TABLE munki_deployments
    ADD COLUMN package_id BIGINT REFERENCES munki_packages (id) ON DELETE CASCADE,
    ADD COLUMN intent munki_deployment_intent;

UPDATE munki_deployments d
SET
    package_id = COALESCE(
        d.pinned_package_id,
        (
            SELECT p.id
            FROM munki_packages p
            WHERE p.software_id = d.software_id
            ORDER BY lower(p.version) DESC, p.id DESC
            LIMIT 1
        )
    ),
    intent = CASE
        WHEN d.self_service = 'featured' THEN 'featured'::munki_deployment_intent
        WHEN d.self_service IN ('available', 'default') THEN 'optional'::munki_deployment_intent
        WHEN d.action = 'install' THEN 'ensure_installed'::munki_deployment_intent
        WHEN d.action = 'remove' THEN 'ensure_absent'::munki_deployment_intent
        WHEN d.action = 'update_if_present' THEN 'update_if_present'::munki_deployment_intent
        ELSE 'optional'::munki_deployment_intent
    END;

DELETE FROM munki_deployments
WHERE package_id IS NULL;

ALTER TABLE munki_deployments
    ALTER COLUMN package_id SET NOT NULL,
    ALTER COLUMN intent SET NOT NULL;

DROP INDEX IF EXISTS munki_deployments_software_idx;
DROP INDEX IF EXISTS munki_deployments_pinned_package_idx;
DROP INDEX IF EXISTS munki_deployments_position_idx;

ALTER TABLE munki_deployments
    DROP CONSTRAINT IF EXISTS munki_deployments_package_selection_check,
    DROP CONSTRAINT IF EXISTS munki_deployments_pinned_package_software_fkey,
    DROP COLUMN software_id,
    DROP COLUMN action,
    DROP COLUMN self_service,
    DROP COLUMN package_selection,
    DROP COLUMN pinned_package_id;

ALTER TABLE munki_packages
    DROP CONSTRAINT IF EXISTS munki_packages_software_id_id_key;

DROP TYPE munki_deployment_action;
DROP TYPE munki_self_service_mode;
DROP TYPE munki_package_selection;

CREATE INDEX munki_deployments_package_idx
    ON munki_deployments (package_id);
CREATE INDEX munki_deployments_position_idx
    ON munki_deployments (position, id);
