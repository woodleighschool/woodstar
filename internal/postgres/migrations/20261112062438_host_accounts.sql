-- +goose Up
DROP TABLE host_users;

CREATE TABLE host_accounts (
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    uid TEXT NOT NULL,
    username TEXT NOT NULL,
    uuid TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    home_directory TEXT NOT NULL DEFAULT '',
    shell TEXT NOT NULL DEFAULT '',
    is_admin BOOLEAN NOT NULL,
    is_hidden BOOLEAN NOT NULL,
    logged_in BOOLEAN NOT NULL,
    filevault_unlock BOOLEAN,
    secure_token BOOLEAN,
    volume_owner BOOLEAN,
    PRIMARY KEY (host_id, uid, username)
);
