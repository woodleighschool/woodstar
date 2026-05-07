-- name: UserExists :one
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE deleted_at IS NULL
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
RETURNING
    id,
    email,
    name,
    password_hash,
    role,
    created_at,
    updated_at,
    deleted_at;

-- name: GetUserByEmail :one
SELECT
    id,
    email,
    name,
    password_hash,
    role,
    created_at,
    updated_at,
    deleted_at
FROM users
WHERE email = @email AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT
    id,
    email,
    name,
    password_hash,
    role,
    created_at,
    updated_at,
    deleted_at
FROM users
WHERE id = @id AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT
    id,
    email,
    name,
    password_hash,
    role,
    created_at,
    updated_at,
    deleted_at
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at;

-- name: UpdateUser :one
UPDATE users
SET
    name = @name,
    role = @role::user_role,
    password_hash = COALESCE(sqlc.narg(password_hash), password_hash),
    updated_at = now()
WHERE id = @id AND deleted_at IS NULL
RETURNING
    id,
    email,
    name,
    password_hash,
    role,
    created_at,
    updated_at,
    deleted_at;

-- name: SoftDeleteUser :one
UPDATE users
SET deleted_at = now(), updated_at = now()
WHERE id = @id AND deleted_at IS NULL
RETURNING id;

-- name: CountAdminUsers :one
SELECT count(*)::integer
FROM users
WHERE role = 'admin' AND deleted_at IS NULL;
