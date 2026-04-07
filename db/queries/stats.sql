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

-- name: GetGlobalLeaderboard :many
WITH user_streaks AS (
    SELECT user_id,
        date,
        date - (ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY date DESC))::int * INTERVAL '1 day' AS grp
    FROM daily_stats
    WHERE reviews > 0
),
current_streaks AS (
    SELECT user_id, COUNT(*)::int AS streak
    FROM user_streaks
    WHERE grp = (
        SELECT grp FROM user_streaks us2
        WHERE us2.user_id = user_streaks.user_id
        ORDER BY date DESC LIMIT 1
    )
    GROUP BY user_id
)
SELECT cs.user_id, u.display_name, cs.streak
FROM current_streaks cs
JOIN users u ON u.id = cs.user_id
WHERE cs.streak > 0
ORDER BY cs.streak DESC, u.display_name
LIMIT $1;
