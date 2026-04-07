-- name: CreateReviewLog :one
INSERT INTO review_logs (card_id, user_id, rating, state, scheduled_days, elapsed_days, stability, difficulty, duration_ms, source, reviewed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListReviewLogsByCardID :many
SELECT * FROM review_logs WHERE card_id = $1 ORDER BY reviewed_at DESC;

-- name: ListReviewLogsByUserID :many
SELECT * FROM review_logs
WHERE user_id = $1 AND reviewed_at >= $2 AND reviewed_at < $3
ORDER BY reviewed_at DESC
LIMIT $4 OFFSET $5;

-- name: CountReviewLogsByUserID :one
SELECT COUNT(*)::int FROM review_logs
WHERE user_id = $1 AND reviewed_at >= $2 AND reviewed_at < $3;

-- name: CountReviewsByUserAndDate :one
SELECT COUNT(*)::int FROM review_logs
WHERE user_id = @user_id AND reviewed_at::date = @date::date;

-- name: GetReviewCountsPerDay :many
SELECT
    reviewed_at::date AS date,
    COUNT(*)::int AS count,
    COUNT(*) FILTER (WHERE rating >= 3)::int AS correct
FROM review_logs
WHERE user_id = $1 AND reviewed_at >= $2 AND reviewed_at < $3
GROUP BY reviewed_at::date
ORDER BY date;
