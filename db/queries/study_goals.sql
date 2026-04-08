-- name: UpsertStudyGoal :one
INSERT INTO study_goals (user_id, goal_type, target_value, is_active)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, goal_type) DO UPDATE SET
    target_value = EXCLUDED.target_value,
    is_active = EXCLUDED.is_active,
    updated_at = now()
RETURNING *;

-- name: GetStudyGoalByID :one
SELECT * FROM study_goals WHERE id = $1;

-- name: ListStudyGoalsByUserID :many
SELECT * FROM study_goals
WHERE user_id = $1 AND is_active = true
ORDER BY created_at;

-- name: DeleteStudyGoal :exec
DELETE FROM study_goals WHERE id = $1;
