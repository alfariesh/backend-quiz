package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type GoalRepository struct {
	db *pgxpool.Pool
}

func NewGoalRepository(db *pgxpool.Pool) *GoalRepository {
	return &GoalRepository{db: db}
}

func (r *GoalRepository) Upsert(ctx context.Context, goal *domain.StudyGoal) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO study_goals (user_id, goal_type, target_value, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, goal_type) DO UPDATE SET
			target_value = EXCLUDED.target_value,
			is_active = EXCLUDED.is_active,
			updated_at = now()
		RETURNING id, created_at, updated_at`,
		goal.UserID, goal.GoalType, goal.TargetValue, goal.IsActive,
	).Scan(&goal.ID, &goal.CreatedAt, &goal.UpdatedAt)
}

func (r *GoalRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.StudyGoal, error) {
	var g domain.StudyGoal
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, goal_type, target_value, is_active, created_at, updated_at
		FROM study_goals WHERE id = $1`, id,
	).Scan(&g.ID, &g.UserID, &g.GoalType, &g.TargetValue, &g.IsActive, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &g, err
}

func (r *GoalRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.StudyGoal, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, goal_type, target_value, is_active, created_at, updated_at
		FROM study_goals WHERE user_id = $1 AND is_active = true
		ORDER BY created_at`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []domain.StudyGoal
	for rows.Next() {
		var g domain.StudyGoal
		if err := rows.Scan(&g.ID, &g.UserID, &g.GoalType, &g.TargetValue, &g.IsActive, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

func (r *GoalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM study_goals WHERE id = $1`, id)
	return err
}
