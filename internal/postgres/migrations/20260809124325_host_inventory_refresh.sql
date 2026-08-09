-- +goose Up
ALTER TABLE hosts
    ADD COLUMN inventory_refresh_requested BOOLEAN NOT NULL DEFAULT FALSE;
