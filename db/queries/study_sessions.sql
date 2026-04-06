-- name: CreateStudySession :one
INSERT INTO study_sessions (user_id, deck_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetStudySessionByID :one
SELECT * FROM study_sessions WHERE id = $1;

-- name: UpdateStudySession :one
UPDATE study_sessions SET
    ended_at = COALESCE(sqlc.narg('ended_at'), ended_at),
    new_count = COALESCE(sqlc.narg('new_count'), new_count),
    review_count = COALESCE(sqlc.narg('review_count'), review_count),
    relearn_count = COALESCE(sqlc.narg('relearn_count'), relearn_count),
    total_duration_ms = COALESCE(sqlc.narg('total_duration_ms'), total_duration_ms)
WHERE id = $1
RETURNING *;

-- name: IncrementSessionCounters :exec
UPDATE study_sessions SET
    new_count = new_count + $2,
    review_count = review_count + $3,
    relearn_count = relearn_count + $4,
    total_duration_ms = total_duration_ms + $5
WHERE id = $1;

-- name: EndStudySession :one
UPDATE study_sessions SET ended_at = now() WHERE id = $1 RETURNING *;

-- name: ListStudySessionsByUserID :many
SELECT * FROM study_sessions
WHERE user_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStudySessionsByUserID :one
SELECT COUNT(*)::int FROM study_sessions WHERE user_id = $1;
