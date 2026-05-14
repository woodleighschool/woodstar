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
    role
)
VALUES (
    @email,
    @name,
    @password_hash,
    @role
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = @email;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = @id;

-- name: ListUsers :many
SELECT *
FROM users
ORDER BY created_at;

-- name: UpdateUser :one
UPDATE users
SET
    name = @name,
    role = @role::user_role,
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
WHERE api_key = @api_key;

-- name: SetUserAPIKey :one
UPDATE users
SET
    api_key = @api_key,
    api_key_created_at = now(),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: ClearUserAPIKey :one
UPDATE users
SET
    api_key = NULL,
    api_key_created_at = NULL,
    updated_at = now()
WHERE id = @id
RETURNING *;
