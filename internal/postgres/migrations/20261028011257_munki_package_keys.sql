-- +goose Up

ALTER TABLE munki_packages DROP COLUMN suppress_bundle_relocation;

ALTER TABLE munki_packages
    ADD COLUMN restart_action_for_uninstall TEXT NOT NULL DEFAULT ''
        CONSTRAINT munki_packages_restart_action_for_uninstall_check CHECK (
            restart_action_for_uninstall IN (
                '', 'None', 'RequireLogout', 'RecommendRestart', 'RequireRestart', 'RequireShutdown'
            )
        ),
    ADD COLUMN blocking_applications_with_launch_args JSONB NOT NULL DEFAULT '[]'::JSONB
        CHECK (jsonb_typeof(blocking_applications_with_launch_args) = 'array'),
    ADD COLUMN download_on_low_data TEXT NOT NULL DEFAULT 'auto'
        CONSTRAINT munki_packages_download_on_low_data_check CHECK (
            download_on_low_data IN ('auto', 'always', 'never')
        ),
    ADD COLUMN custom_action_labels JSONB NOT NULL DEFAULT '{}'::JSONB
        CHECK (jsonb_typeof(custom_action_labels) = 'object'),
    ADD CONSTRAINT munki_packages_blocking_applications_launch_args_check CHECK (
        NOT blocking_applications_none OR blocking_applications_with_launch_args = '[]'::JSONB
    );
