-- name: ListEnrollSecrets :many
SELECT *
FROM enroll_secrets
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CreateEnrollSecret :one
INSERT INTO enroll_secrets (
    value
)
VALUES (
    @value
)
RETURNING *;

-- name: HasActiveEnrollSecret :one
SELECT EXISTS (
    SELECT 1
    FROM enroll_secrets
    WHERE value = @value
      AND deleted_at IS NULL
);

-- name: DeleteEnrollSecret :one
UPDATE enroll_secrets
SET deleted_at = now()
WHERE id = @id
  AND deleted_at IS NULL
RETURNING id;
