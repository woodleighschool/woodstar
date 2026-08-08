-- +goose Up

ALTER TABLE hosts
    ADD COLUMN software_inventory_updated_at TIMESTAMPTZ;

UPDATE hosts
SET software_inventory_updated_at = inventory_updated_at;

ALTER TABLE host_software_installed_paths
    ADD COLUMN bundle_short_version TEXT NOT NULL DEFAULT '',
    ADD COLUMN bundle_version TEXT NOT NULL DEFAULT '';

ALTER TABLE munki_software
    ADD COLUMN installation_detector_bundle_identifier TEXT,
    ADD COLUMN installation_detector_expected_path TEXT,
    ADD COLUMN installation_detector_version_source TEXT,
    ADD COLUMN installation_detector_automatic BOOLEAN NOT NULL DEFAULT FALSE,
    ADD CONSTRAINT munki_software_installation_detector_check CHECK (
        (
            installation_detector_bundle_identifier IS NULL
            AND installation_detector_expected_path IS NULL
            AND installation_detector_version_source IS NULL
            AND NOT installation_detector_automatic
        )
        OR (
            installation_detector_bundle_identifier IS NOT NULL
            AND installation_detector_version_source IS NOT NULL
            AND btrim(installation_detector_bundle_identifier) <> ''
            AND installation_detector_version_source IN (
                'bundle_short_version',
                'bundle_version'
            )
        )
    );

WITH detector_candidates AS (
    SELECT
        p.software_id,
        btrim(COALESCE(item ->> 'bundle_identifier', '')) AS bundle_identifier,
        btrim(COALESCE(item ->> 'path', '')) AS expected_path,
        CASE
            WHEN btrim(COALESCE(item ->> 'version_comparison_key', '')) = 'CFBundleShortVersionString'
                THEN 'bundle_short_version'
            WHEN btrim(COALESCE(item ->> 'version_comparison_key', '')) = 'CFBundleVersion'
                THEN 'bundle_version'
            WHEN btrim(COALESCE(item ->> 'version_comparison_key', '')) = ''
                AND btrim(COALESCE(item ->> 'bundle_short_version', '')) <> ''
                AND btrim(COALESCE(item ->> 'bundle_version', '')) = ''
                THEN 'bundle_short_version'
            WHEN btrim(COALESCE(item ->> 'version_comparison_key', '')) = ''
                AND btrim(COALESCE(item ->> 'bundle_short_version', '')) = ''
                AND btrim(COALESCE(item ->> 'bundle_version', '')) <> ''
                THEN 'bundle_version'
        END AS version_source
    FROM munki_packages p
    CROSS JOIN LATERAL jsonb_array_elements(p.installs) AS item
    WHERE item ->> 'type' = 'application'
      AND p.installer_type <> 'nopkg'
), resolved_detectors AS (
    SELECT
        software_id,
        min(bundle_identifier) AS bundle_identifier,
        CASE
            WHEN count(*) = count(NULLIF(expected_path, ''))
                AND count(DISTINCT expected_path) = 1
                THEN min(expected_path)
        END AS expected_path,
        min(version_source) AS version_source
    FROM detector_candidates
    GROUP BY software_id
    HAVING bool_and(bundle_identifier <> '' AND version_source IS NOT NULL)
       AND count(DISTINCT bundle_identifier) = 1
       AND count(DISTINCT version_source) = 1
)
UPDATE munki_software software
SET
    installation_detector_bundle_identifier = detector.bundle_identifier,
    installation_detector_expected_path = detector.expected_path,
    installation_detector_version_source = detector.version_source,
    installation_detector_automatic = TRUE
FROM resolved_detectors detector
WHERE software.id = detector.software_id;
