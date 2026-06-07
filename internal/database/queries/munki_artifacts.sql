-- name: UpsertMunkiArtifact :one
INSERT INTO munki_artifacts (
    kind,
    display_name,
    location,
    content_type,
    size_bytes,
    sha256,
    storage_key
)
VALUES (
    @kind::munki_artifact_kind,
    @display_name,
    @location,
    @content_type,
    @size_bytes,
    @sha256,
    @storage_key
)
ON CONFLICT (kind, location) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    content_type = EXCLUDED.content_type,
    size_bytes = EXCLUDED.size_bytes,
    sha256 = EXCLUDED.sha256,
    storage_key = EXCLUDED.storage_key,
    updated_at = now()
RETURNING *;

-- name: ListMunkiArtifacts :many
SELECT *
FROM munki_artifacts
ORDER BY lower(COALESCE(NULLIF(display_name, ''), location)), lower(location), id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountMunkiArtifacts :one
SELECT COUNT(*)::integer
FROM munki_artifacts;

-- name: GetMunkiArtifactByID :one
SELECT *
FROM munki_artifacts
WHERE id = @id;

-- name: GetMunkiArtifactByKindAndLocation :one
SELECT *
FROM munki_artifacts
WHERE kind = @kind::munki_artifact_kind
  AND location = @location;
