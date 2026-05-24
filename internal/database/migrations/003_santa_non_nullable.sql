-- +goose Up

-- Santa configuration knobs become non-nullable. The admin SPA defaults the
-- form to Santa's own defaults and sends explicit values, so the backend no
-- longer carries hidden defaults or tri-state plumbing for these columns.
--
-- The removable_media policy stays nullable at the column level: an empty
-- action means "no policy".

UPDATE santa_configurations
SET
    enable_bundles                  = COALESCE(enable_bundles, false),
    enable_transitive_rules         = COALESCE(enable_transitive_rules, false),
    enable_all_event_upload         = COALESCE(enable_all_event_upload, false),
    full_sync_interval_seconds      = COALESCE(full_sync_interval_seconds, 600),
    batch_size                      = COALESCE(batch_size, 50),
    allowed_path_regex              = COALESCE(allowed_path_regex, ''),
    blocked_path_regex              = COALESCE(blocked_path_regex, ''),
    event_detail_url                = COALESCE(event_detail_url, ''),
    event_detail_text               = COALESCE(event_detail_text, '');

ALTER TABLE santa_configurations
    ALTER COLUMN client_mode                  DROP DEFAULT,
    ALTER COLUMN enable_bundles               SET NOT NULL,
    ALTER COLUMN enable_transitive_rules      SET NOT NULL,
    ALTER COLUMN enable_all_event_upload      SET NOT NULL,
    ALTER COLUMN full_sync_interval_seconds   SET NOT NULL,
    ALTER COLUMN batch_size                   SET NOT NULL,
    ALTER COLUMN allowed_path_regex           SET NOT NULL,
    ALTER COLUMN blocked_path_regex           SET NOT NULL,
    ALTER COLUMN event_detail_url             SET NOT NULL,
    ALTER COLUMN event_detail_text            SET NOT NULL;

ALTER TABLE santa_configurations
    ADD CONSTRAINT santa_configurations_batch_size_check
        CHECK (batch_size BETWEEN 5 AND 100);
