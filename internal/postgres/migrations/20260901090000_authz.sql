-- +goose Up

CREATE TABLE authz_roles (
    id BIGSERIAL PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    builtin BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE authz_role_permissions (
    role_id BIGINT NOT NULL REFERENCES authz_roles (id) ON DELETE CASCADE,
    resource TEXT NOT NULL,
    access SMALLINT NOT NULL CHECK (access IN (1, 2)),
    PRIMARY KEY (role_id, resource)
);

CREATE TABLE authz_user_roles (
	user_id BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
	role_id BIGINT NOT NULL REFERENCES authz_roles (id) ON DELETE CASCADE
);

CREATE TABLE authz_group_roles (
	group_id BIGINT PRIMARY KEY REFERENCES directory_groups (id) ON DELETE CASCADE,
	role_id BIGINT NOT NULL REFERENCES authz_roles (id) ON DELETE CASCADE
);

INSERT INTO authz_roles (key, name, description, builtin)
VALUES
    ('admin', 'Admin', 'Full access to every resource.', true),
    ('viewer', 'Viewer', 'Read-only access to ordinary application resources.', true);

INSERT INTO authz_role_permissions (role_id, resource, access)
SELECT role.id, resource, 2
FROM authz_roles AS role
CROSS JOIN unnest(ARRAY[
    'activity',
    'users',
    'groups',
    'directory',
    'hosts',
    'labels',
    'software',
    'agents.secrets',
    'munki.software',
    'munki.packages',
    'munki.distribution-points',
    'munki.client-resources',
    'osquery.overview',
    'osquery.reports',
    'osquery.policies',
    'osquery.live-queries',
    'osquery.remediations',
    'santa.configurations',
    'santa.events',
    'santa.rules'
]) AS resource
WHERE role.key = 'admin';

INSERT INTO authz_role_permissions (role_id, resource, access)
SELECT role.id, resource, 1
FROM authz_roles AS role
CROSS JOIN unnest(ARRAY[
    'activity',
    'users',
    'groups',
    'directory',
    'hosts',
    'labels',
    'software',
    'munki.software',
    'munki.packages',
    'munki.distribution-points',
    'munki.client-resources',
    'osquery.overview',
    'osquery.reports',
    'osquery.policies',
    'santa.configurations',
    'santa.events',
    'santa.rules'
]) AS resource
WHERE role.key = 'viewer';

INSERT INTO authz_user_roles (user_id, role_id)
SELECT users.id, roles.id
FROM users
JOIN authz_roles AS roles ON roles.key = users.role::text
WHERE users.role IS NOT NULL;

ALTER TABLE users DROP COLUMN role;
DROP TYPE user_role;
