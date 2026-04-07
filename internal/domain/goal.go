package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	GoalTypeDailyReviews  = "daily_reviews"
	GoalTypeDailyNew      = "daily_new"
	GoalTypeWeeklyReviews = "weekly_reviews"
	GoalTypeDailyMinutes  = "daily_minutes"
)

type StudyGoal struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	GoalType    string    `json:"goal_type"`
	TargetValue int       `json:"target_value"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GoalRepository interface {
	Upsert(ctx context.Context, goal *StudyGoal) error
	GetByID(ctx context.Context, id uuid.UUID) (*StudyGoal, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]StudyGoal, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
