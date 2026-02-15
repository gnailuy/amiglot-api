-- name: GetUserByID :one
SELECT id, email, created_at, last_login_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, created_at, last_login_at
FROM users
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (email)
VALUES ($1)
RETURNING id, email, created_at, last_login_at;

-- name: CreateMagicLinkToken :one
INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, consumed_at, created_at;

-- name: GetValidMagicLinkToken :one
SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
FROM magic_link_tokens
WHERE token_hash = $1
  AND consumed_at IS NULL
  AND expires_at > now();

-- name: ConsumeMagicLinkToken :exec
UPDATE magic_link_tokens
SET consumed_at = now()
WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = now()
WHERE id = $1;
