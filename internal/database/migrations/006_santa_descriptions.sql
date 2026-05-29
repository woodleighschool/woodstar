-- +goose Up

ALTER TABLE santa_configurations
    ADD COLUMN description TEXT NOT NULL DEFAULT '';

ALTER TABLE santa_rules
    ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE santa_rules
    DROP COLUMN IF EXISTS description;

ALTER TABLE santa_configurations
    DROP COLUMN IF EXISTS description;
