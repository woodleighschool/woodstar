-- +goose Up

DROP INDEX hosts_active_seen_idx;
ALTER TABLE hosts
    DROP COLUMN last_seen_at,
    DROP COLUMN last_remote_ip;
ALTER TABLE santa_hosts
    DROP COLUMN last_seen_at;
