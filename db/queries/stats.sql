-- name: UpsertDailyStats :one
INSERT INTO daily_stats (user_id, date, new_cards, reviews, relearns, total_duration_ms, retention_rate)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, date) DO UPDATE SET
    new_cards = EXCLUDED.new_cards,
    reviews = EXCLUDED.reviews,
    relearns = EXCLUDED.relearns,
    total_duration_ms = EXCLUDED.total_duration_ms,
    retention_rate = EXCLUDED.retention_rate
RETURNING *;

-- name: GetDailyStats :many
SELECT * FROM daily_stats
WHERE user_id = $1 AND date >= $2 AND date <= $3
ORDER BY date;

-- name: GetStreak :one
WITH streak AS (
    SELECT date,
        date - (ROW_NUMBER() OVER (ORDER BY date DESC))::int * INTERVAL '1 day' AS grp
    FROM daily_stats
    WHERE user_id = $1 AND reviews > 0
)
SELECT COALESCE(COUNT(*)::int, 0) AS streak
FROM streak
WHERE grp = (SELECT grp FROM streak LIMIT 1);

-- name: GetTotalReviews :one
SELECT COALESCE(SUM(reviews)::int, 0) AS total FROM daily_stats WHERE user_id = $1;

-- name: GetAverageRetention :one
SELECT COALESCE(AVG(retention_rate), 0)::real AS avg_retention
FROM daily_stats
WHERE user_id = $1 AND retention_rate IS NOT NULL AND date >= $2;
