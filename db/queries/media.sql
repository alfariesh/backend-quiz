-- name: CreateMedia :one
INSERT INTO media (user_id, card_id, file_name, file_size, mime_type, r2_key, url)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetMediaByID :one
SELECT * FROM media WHERE id = $1;

-- name: DeleteMedia :exec
DELETE FROM media WHERE id = $1;

-- name: ListMediaByCardID :many
SELECT * FROM media WHERE card_id = $1 ORDER BY created_at;

-- name: ListMediaByUserID :many
SELECT * FROM media WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;
