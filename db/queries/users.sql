-- name: CreateUser :one
INSERT INTO users (email, password_hash, display_name, timezone, desired_retention, daily_new_limit, daily_review_limit)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users SET
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    timezone = COALESCE(sqlc.narg('timezone'), timezone),
    desired_retention = COALESCE(sqlc.narg('desired_retention'), desired_retention),
    daily_new_limit = COALESCE(sqlc.narg('daily_new_limit'), daily_new_limit),
    daily_review_limit = COALESCE(sqlc.narg('daily_review_limit'), daily_review_limit),
    fsrs_weights = COALESCE(sqlc.narg('fsrs_weights'), fsrs_weights),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: CreateOAuthAccount :one
INSERT INTO oauth_accounts (user_id, provider, provider_id, email, avatar_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetOAuthAccount :one
SELECT * FROM oauth_accounts WHERE provider = $1 AND provider_id = $2;

-- name: GetOAuthAccountsByUserID :many
SELECT * FROM oauth_accounts WHERE user_id = $1;
