-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3,
    $4
)
RETURNING token;

-- name: GetRefreshToken :one
SELECT 
    *
FROM
    refresh_tokens
WHERE
    token = $1;

-- name: SetRevoked :exec
UPDATE 
    refresh_tokens
SET 
    revoked_at = $1
WHERE
    token = $2;

-- name: GetUserFromRefreshToken :one
SELECT
    *
FROM
    users
JOIN refresh_tokens ON refresh_tokens.user_id = users.id
WHERE 
    token = $1;