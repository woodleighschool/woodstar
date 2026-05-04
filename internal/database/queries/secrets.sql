-- name: ListSecrets :many
SELECT
    id,
    kind,
    value,
    created_at,
    deleted_at
FROM secrets
WHERE kind = @kind AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CreateSecret :one
INSERT INTO secrets (
    kind,
    value
)
VALUES (
    @kind,
    @value
)
RETURNING
    id,
    kind,
    value,
    created_at,
    deleted_at;

-- name: HasActiveSecret :one
SELECT EXISTS (
    SELECT 1
    FROM secrets
    WHERE kind = @kind
      AND value = @value
      AND deleted_at IS NULL
);

-- name: DeleteSecret :one
UPDATE secrets
SET deleted_at = now()
WHERE id = @id
  AND kind = @kind
  AND deleted_at IS NULL
RETURNING id;
