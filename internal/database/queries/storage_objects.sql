-- name: CreateStorageObject :one
INSERT INTO storage_objects (prefix, filename, content_type)
VALUES (@prefix, @filename, @content_type)
RETURNING *;

-- name: ConfirmStorageObject :one
UPDATE storage_objects
SET size_bytes = @size_bytes,
    sha256 = @sha256,
    content_type = COALESCE(NULLIF(@content_type::text, ''), content_type),
    available_at = now(),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: GetStorageObjectByID :one
SELECT *
FROM storage_objects
WHERE id = @id;

-- name: ListStorageObjectsByPrefix :many
SELECT *
FROM storage_objects
WHERE prefix = @prefix
  AND available_at IS NOT NULL
ORDER BY created_at DESC, id DESC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountStorageObjectsByPrefix :one
SELECT COUNT(*)::integer
FROM storage_objects
WHERE prefix = @prefix
  AND available_at IS NOT NULL;

-- name: DeleteStorageObject :execrows
DELETE FROM storage_objects
WHERE id = @id;

-- name: ListPendingStorageObjectsBefore :many
SELECT *
FROM storage_objects
WHERE available_at IS NULL
  AND created_at < @cutoff
ORDER BY id;
