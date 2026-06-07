-- name: ListAgentSecrets :many
SELECT *
FROM agent_secrets
WHERE deleted_at IS NULL
ORDER BY agent ASC, created_at DESC, id DESC;

-- name: CreateAgentSecret :one
INSERT INTO agent_secrets (
    agent,
    value
)
VALUES (
    @agent,
    @value
)
RETURNING *;

-- name: HasActiveAgentSecret :one
SELECT EXISTS (
    SELECT 1
    FROM agent_secrets
    WHERE agent = @agent
      AND value = @value
      AND deleted_at IS NULL
);

-- name: UpdateAgentSecret :one
UPDATE agent_secrets
SET value = @value
WHERE id = @id
  AND deleted_at IS NULL
RETURNING *;

-- name: DeleteAgentSecret :one
UPDATE agent_secrets
SET deleted_at = now()
WHERE id = @id
  AND deleted_at IS NULL
RETURNING id;
