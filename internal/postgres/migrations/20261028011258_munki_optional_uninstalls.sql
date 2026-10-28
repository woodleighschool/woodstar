-- +goose Up

ALTER TYPE munki_manifest_action ADD VALUE 'optional_uninstalls' AFTER 'optional_installs';
