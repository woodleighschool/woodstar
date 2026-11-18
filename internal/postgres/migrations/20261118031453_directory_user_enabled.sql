-- +goose Up

ALTER TABLE users ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;

-- A snapshot can move identifiers between objects in a single statement.
-- Deferrable constraints check the resulting directory after those moves.
DROP INDEX users_active_provider_email_idx;
DROP INDEX users_active_provider_upn_idx;

ALTER TABLE users
    ADD CONSTRAINT users_provider_signin_email
        EXCLUDE USING btree (email WITH =)
        WHERE (source <> 'local' AND deleted_at IS NULL AND enabled)
        DEFERRABLE INITIALLY IMMEDIATE,
    ADD CONSTRAINT users_provider_current_upn
        EXCLUDE USING btree (user_principal_name WITH =)
        WHERE (source <> 'local' AND deleted_at IS NULL)
        DEFERRABLE INITIALLY IMMEDIATE;

CREATE VIEW host_primary_users AS
SELECT p.host_id, p.email, p.source, u.id AS user_id, u.mail_nickname, u.name, u.department
FROM (
    -- Source enum order gives a manual assignment precedence over Orbit.
    SELECT DISTINCT ON (host_id) host_id, email, source
    FROM host_primary_user_sources
    ORDER BY host_id, source
) AS p
LEFT JOIN LATERAL (
    SELECT id, mail_nickname, name, department
    FROM users
    WHERE deleted_at IS NULL AND (email = p.email OR user_principal_name = p.email)
    ORDER BY (email = p.email) DESC, (source <> 'local') DESC, enabled DESC, id
    LIMIT 1
) AS u ON true;
