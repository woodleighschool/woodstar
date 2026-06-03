-- name: UserExists :one
SELECT EXISTS (
    SELECT 1
    FROM users
);

-- name: CreateUser :one
INSERT INTO users (
    email,
    name,
    password_hash,
    role,
    active
)
VALUES (
    @email,
    @name,
    @password_hash,
    @role::user_role,
    true
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE lower(email) = lower(@email)
   OR lower(COALESCE(user_principal_name, '')) = lower(@email)
ORDER BY CASE WHEN lower(email) = lower(@email) THEN 0 ELSE 1 END, id
LIMIT 1;

-- name: GetLoginUserByEmail :one
SELECT *
FROM users
WHERE active
  AND role IS NOT NULL
  AND password_hash IS NOT NULL
  AND (
      lower(email) = lower(@email)
      OR lower(COALESCE(user_principal_name, '')) = lower(@email)
  )
ORDER BY CASE WHEN lower(email) = lower(@email) THEN 0 ELSE 1 END, id
LIMIT 1;

-- name: GetSSOUserByEmail :one
SELECT *
FROM users
WHERE active
  AND role IS NOT NULL
  AND (
      lower(email) = lower(@email)
      OR lower(COALESCE(user_principal_name, '')) = lower(@email)
  )
ORDER BY CASE WHEN lower(email) = lower(@email) THEN 0 ELSE 1 END, id
LIMIT 1;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = @id;

-- name: ListUsers :many
SELECT *
FROM users
ORDER BY lower(name), id;

-- name: UpdateUser :one
UPDATE users
SET
    name = CASE WHEN entra_id IS NULL THEN @name ELSE name END,
    role = sqlc.narg(role)::user_role,
    password_hash = COALESCE(sqlc.narg(password_hash), password_hash),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: UpdateAccountByID :one
UPDATE users
SET
    name = CASE WHEN entra_id IS NULL THEN @name ELSE name END,
    password_hash = COALESCE(sqlc.narg(password_hash), password_hash),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteUser :one
DELETE FROM users
WHERE id = @id
RETURNING id;

-- name: GetUserByAPIKey :one
SELECT *
FROM users
WHERE api_key = @api_key
  AND active
  AND role IS NOT NULL;

-- name: SetUserAPIKey :one
UPDATE users
SET
    api_key = @api_key,
    api_key_created_at = now(),
    updated_at = now()
WHERE id = @id
  AND active
  AND role IS NOT NULL
RETURNING *;

-- name: ClearUserAPIKey :one
UPDATE users
SET
    api_key = NULL,
    api_key_created_at = NULL,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: AttachEntraUserByEmail :exec
UPDATE users
SET
    entra_id = @external_id::text,
    updated_at = now()
WHERE entra_id IS NULL
  AND lower(email) = lower(COALESCE(sqlc.narg(mail)::text, @user_principal_name::text));

-- name: UpsertEntraUser :one
INSERT INTO users (
    email,
    name,
    entra_id,
    user_principal_name,
    mail_nickname,
    given_name,
    family_name,
    department,
    active,
    last_synced_at
)
VALUES (
    COALESCE(sqlc.narg(mail)::text, @user_principal_name::text),
    @display_name::text,
    @external_id::text,
    @user_principal_name::text,
    sqlc.narg(mail_nickname)::text,
    sqlc.narg(given_name)::text,
    sqlc.narg(family_name)::text,
    sqlc.narg(department)::text,
    @active::boolean,
    @last_synced_at::timestamptz
)
ON CONFLICT (entra_id) DO UPDATE SET
    email = EXCLUDED.email,
    name = EXCLUDED.name,
    user_principal_name = EXCLUDED.user_principal_name,
    mail_nickname = EXCLUDED.mail_nickname,
    given_name = EXCLUDED.given_name,
    family_name = EXCLUDED.family_name,
    department = EXCLUDED.department,
    active = EXCLUDED.active,
    last_synced_at = EXCLUDED.last_synced_at,
    updated_at = now()
RETURNING *;

-- name: MarkEntraUsersInactiveNotIn :exec
UPDATE users
SET
    active = false,
    updated_at = now()
WHERE entra_id IS NOT NULL
  AND entra_id <> ALL(@external_ids::text[]);
