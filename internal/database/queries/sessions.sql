-- name: CreateSession :exec
INSERT INTO sessions (
    user_id,
    token_hash,
    expires_at
)
VALUES (
    @user_id,
    @token_hash,
    @expires_at
);

-- name: GetUserByActiveSessionToken :one
WITH refreshed_session AS (
    UPDATE sessions
    SET last_seen_at = @seen_at
    WHERE token_hash = @token_hash
      AND revoked_at IS NULL
      AND expires_at > @seen_at
    RETURNING user_id
)
SELECT
    users.id,
    users.email,
    users.name,
    users.password_hash,
    users.role,
    users.created_at,
    users.updated_at,
    users.deleted_at
FROM users
JOIN refreshed_session ON refreshed_session.user_id = users.id
WHERE users.deleted_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = now()
WHERE token_hash = @token_hash AND revoked_at IS NULL;

-- name: RevokeSessionsForUser :exec
UPDATE sessions
SET revoked_at = now()
WHERE user_id = @user_id AND revoked_at IS NULL;
