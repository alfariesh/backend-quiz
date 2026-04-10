package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/repository/sqlc"
)

type GoalRepository struct {
	q *sqlc.Queries
}

func NewGoalRepository(pool *pgxpool.Pool) *GoalRepository {
	return &GoalRepository{q: sqlc.New(pool)}
}

func (r *GoalRepository) Upsert(ctx context.Context, goal *domain.StudyGoal) error {
	result, err := querier(r.q, ctx).UpsertStudyGoal(ctx, sqlc.UpsertStudyGoalParams{
		UserID:      goal.UserID,
		GoalType:    goal.GoalType,
		TargetValue: int32(goal.TargetValue),
		IsActive:    goal.IsActive,
	})
	if err != nil {
		return err
	}
	*goal = goalFromSqlc(result)
	return nil
}

func (r *GoalRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.StudyGoal, error) {
	result, err := querier(r.q, ctx).GetStudyGoalByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	g := goalFromSqlc(result)
	return &g, nil
}

func (r *GoalRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.StudyGoal, error) {
	rows, err := querier(r.q, ctx).ListStudyGoalsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	goals := make([]domain.StudyGoal, len(rows))
	for i, row := range rows {
		goals[i] = goalFromSqlc(row)
	}
	return goals, nil
}

func (r *GoalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return querier(r.q, ctx).DeleteStudyGoal(ctx, id)
}
