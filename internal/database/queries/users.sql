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
    source
)
VALUES (
    @email,
    @name,
    @password_hash,
    @role::user_role,
    'local'
)
RETURNING *;

-- name: GetLoginUserByEmail :one
SELECT *
FROM users
WHERE deleted_at IS NULL
  AND source = 'local'
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
WHERE deleted_at IS NULL
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
WHERE id = @id
  AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT *
FROM users
ORDER BY lower(name), id;

-- name: UpdateUser :one
UPDATE users
SET
    name = CASE WHEN source = 'local' THEN @name ELSE name END,
    role = sqlc.narg(role)::user_role,
    password_hash = CASE
        WHEN source = 'local' THEN COALESCE(sqlc.narg(password_hash), password_hash)
        ELSE password_hash
    END,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: UpdateAccountByID :one
UPDATE users
SET
    name = CASE WHEN source = 'local' THEN @name ELSE name END,
    password_hash = CASE
        WHEN source = 'local' THEN COALESCE(sqlc.narg(password_hash), password_hash)
        ELSE password_hash
    END,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteUser :one
DELETE FROM users
WHERE id = @id
  AND source = 'local'
  AND deleted_at IS NULL
RETURNING id;

-- name: SoftDeleteDirectoryUser :one
UPDATE users
SET
    deleted_at = now(),
    updated_at = now()
WHERE id = @id
  AND source <> 'local'
  AND deleted_at IS NULL
RETURNING *;

-- name: GetUserByAPIKey :one
SELECT *
FROM users
WHERE api_key = @api_key
  AND deleted_at IS NULL
  AND role IS NOT NULL;

-- name: SetUserAPIKey :one
UPDATE users
SET
    api_key = @api_key,
    api_key_created_at = now(),
    updated_at = now()
WHERE id = @id
  AND deleted_at IS NULL
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

-- name: AttachDirectoryUserByEmail :exec
UPDATE users
SET
    source = @source::directory_source,
    external_id = @external_id::text,
    deleted_at = NULL,
    updated_at = now()
WHERE (
      (source = 'local' AND deleted_at IS NULL)
      OR (source = @source::directory_source AND deleted_at IS NOT NULL)
  )
  AND (
      lower(email) = lower(COALESCE(sqlc.narg(mail)::text, @user_principal_name::text))
      OR lower(COALESCE(user_principal_name, '')) = lower(@user_principal_name::text)
  );

-- name: UpsertDirectoryUser :one
INSERT INTO users (
    email,
    name,
    source,
    external_id,
    user_principal_name,
    mail_nickname,
    given_name,
    family_name,
    department,
    deleted_at
)
VALUES (
    COALESCE(sqlc.narg(mail)::text, @user_principal_name::text),
    @display_name::text,
    @source::directory_source,
    @external_id::text,
    @user_principal_name::text,
    sqlc.narg(mail_nickname)::text,
    sqlc.narg(given_name)::text,
    sqlc.narg(family_name)::text,
    sqlc.narg(department)::text,
    CASE WHEN @enabled::boolean THEN NULL ELSE now() END
)
ON CONFLICT (source, external_id) DO UPDATE SET
    email = EXCLUDED.email,
    name = EXCLUDED.name,
    user_principal_name = EXCLUDED.user_principal_name,
    mail_nickname = EXCLUDED.mail_nickname,
    given_name = EXCLUDED.given_name,
    family_name = EXCLUDED.family_name,
    department = EXCLUDED.department,
    deleted_at = EXCLUDED.deleted_at,
    updated_at = now()
RETURNING *;

-- name: SoftDeleteDirectoryUsersNotIn :exec
UPDATE users
SET
    deleted_at = now(),
    updated_at = now()
WHERE source = @source::directory_source
  AND deleted_at IS NULL
  AND external_id <> ALL(@external_ids::text[]);
