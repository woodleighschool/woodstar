-- +goose Up

ALTER TABLE users DROP CONSTRAINT users_email_key;
ALTER TABLE users DROP CONSTRAINT users_user_principal_name_key;

INSERT INTO users (
    email,
    name,
    source,
    external_id,
    user_principal_name,
    mail_nickname,
    given_name,
    family_name,
    department
)
SELECT
    u.email,
    u.name,
    l.source,
    l.external_id,
    u.user_principal_name,
    u.mail_nickname,
    u.given_name,
    u.family_name,
    u.department
FROM directory_user_links l
JOIN users u ON u.id = l.user_id
WHERE u.source = 'local'
  AND u.deleted_at IS NULL
ON CONFLICT (source, external_id) DO NOTHING;

UPDATE labels l
SET
    criteria = jsonb_set(l.criteria, '{values}', rewritten.values),
    updated_at = now()
FROM (
    SELECT
        target.id,
        jsonb_agg(
            COALESCE(provider.id::text, value.item)
            ORDER BY value.position
        ) AS values
    FROM labels target
    CROSS JOIN LATERAL jsonb_array_elements_text(target.criteria->'values')
        WITH ORDINALITY AS value(item, position)
    LEFT JOIN directory_user_links link
        ON link.user_id::text = value.item
    LEFT JOIN users provider
        ON provider.source = link.source
       AND provider.external_id = link.external_id
    WHERE target.criteria->>'attribute' = 'user'
    GROUP BY target.id
) rewritten
WHERE l.id = rewritten.id;

INSERT INTO directory_group_memberships (user_id, group_id)
SELECT provider.id, membership.group_id
FROM directory_user_links link
JOIN users provider
  ON provider.source = link.source
 AND provider.external_id = link.external_id
JOIN directory_group_memberships membership
  ON membership.user_id = link.user_id
JOIN directory_groups g
  ON g.id = membership.group_id
 AND g.source = link.source
ON CONFLICT DO NOTHING;

DELETE FROM directory_group_memberships gm
USING directory_groups g, directory_user_links l
WHERE gm.user_id = l.user_id
  AND gm.group_id = g.id
  AND g.source = l.source;

UPDATE users
SET
    user_principal_name = NULL,
    mail_nickname = NULL,
    given_name = NULL,
    family_name = NULL,
    department = NULL,
    updated_at = now()
WHERE id IN (SELECT user_id FROM directory_user_links);

DROP TABLE directory_user_links;

CREATE UNIQUE INDEX users_active_local_email_idx
    ON users (email)
    WHERE deleted_at IS NULL AND source = 'local';

CREATE UNIQUE INDEX users_active_provider_email_idx
    ON users (email)
    WHERE deleted_at IS NULL AND source <> 'local';

CREATE UNIQUE INDEX users_active_provider_upn_idx
    ON users (user_principal_name)
    WHERE deleted_at IS NULL AND source <> 'local';
