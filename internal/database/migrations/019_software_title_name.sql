-- +goose Up

ALTER TABLE software_titles
    DROP COLUMN display_name;
