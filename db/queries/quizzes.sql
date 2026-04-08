-- name: CreateQuiz :one
INSERT INTO quizzes (user_id, deck_id, title, description, quiz_type, time_limit_seconds, shuffle_questions)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetQuizByID :one
SELECT * FROM quizzes WHERE id = $1;

-- name: ListQuizzesByUserID :many
SELECT q.*,
    COALESCE(qc.cnt, 0)::int AS question_count,
    COALESCE(ac.cnt, 0)::int AS attempt_count
FROM quizzes q
LEFT JOIN LATERAL (SELECT COUNT(*) AS cnt FROM quiz_questions WHERE quiz_id = q.id) qc ON true
LEFT JOIN LATERAL (SELECT COUNT(*) AS cnt FROM quiz_attempts WHERE quiz_id = q.id) ac ON true
WHERE q.user_id = $1
ORDER BY q.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountQuizzesByUserID :one
SELECT COUNT(*)::int FROM quizzes WHERE user_id = $1;

-- name: UpdateQuiz :exec
UPDATE quizzes SET
    title = $2,
    description = $3,
    quiz_type = $4,
    time_limit_seconds = $5,
    shuffle_questions = $6,
    is_published = $7,
    updated_at = now()
WHERE id = $1;

-- name: DeleteQuiz :exec
DELETE FROM quizzes WHERE id = $1;

-- name: ListQuizzesByDeckAndType :many
SELECT * FROM quizzes
WHERE deck_id = $1 AND quiz_type = $2
ORDER BY created_at;

-- name: CreateQuizQuestion :one
INSERT INTO quiz_questions (quiz_id, card_id, question_type, question_text, options, correct_answer, explanation, position, points)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetQuizQuestionByID :one
SELECT * FROM quiz_questions WHERE id = $1;

-- name: ListQuizQuestionsByQuizID :many
SELECT * FROM quiz_questions WHERE quiz_id = $1
ORDER BY position, created_at;

-- name: UpdateQuizQuestion :exec
UPDATE quiz_questions SET
    question_type = $2,
    question_text = $3,
    options = $4,
    correct_answer = $5,
    explanation = $6,
    position = $7,
    points = $8,
    updated_at = now()
WHERE id = $1;

-- name: DeleteQuizQuestion :exec
DELETE FROM quiz_questions WHERE id = $1;

-- name: CountQuizQuestionsByQuizID :one
SELECT COUNT(*)::int FROM quiz_questions WHERE quiz_id = $1;
