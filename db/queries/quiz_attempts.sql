-- name: CreateQuizAttempt :one
INSERT INTO quiz_attempts (quiz_id, user_id, total_points, total_questions)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetQuizAttemptByID :one
SELECT * FROM quiz_attempts WHERE id = $1;

-- name: UpdateQuizAttempt :exec
UPDATE quiz_attempts SET
    completed_at = $2,
    score = $3,
    correct_count = $4,
    duration_ms = $5
WHERE id = $1;

-- name: ListQuizAttemptsByQuizID :many
SELECT * FROM quiz_attempts WHERE quiz_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: CountQuizAttemptsByQuizID :one
SELECT COUNT(*)::int FROM quiz_attempts WHERE quiz_id = $1;

-- name: ListQuizAttemptsByUserID :many
SELECT * FROM quiz_attempts WHERE user_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: CountQuizAttemptsByUserID :one
SELECT COUNT(*)::int FROM quiz_attempts WHERE user_id = $1;

-- name: GetBestAttemptByQuizID :one
SELECT * FROM quiz_attempts
WHERE quiz_id = $1 AND completed_at IS NOT NULL
ORDER BY score DESC
LIMIT 1;

-- name: CreateQuizAnswer :one
INSERT INTO quiz_answers (attempt_id, question_id, user_answer, is_correct, points_earned, duration_ms)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListQuizAnswersByAttemptID :many
SELECT * FROM quiz_answers WHERE attempt_id = $1
ORDER BY answered_at;

-- name: GetQuizAnswerByAttemptAndQuestion :one
SELECT * FROM quiz_answers
WHERE attempt_id = $1 AND question_id = $2;
