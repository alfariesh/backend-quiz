package dto

import "github.com/rekanesiads/backend-quiz/internal/domain"

type SetGoalRequest struct {
	GoalType    string `json:"goal_type" validate:"required,oneof=daily_reviews daily_new weekly_reviews daily_minutes"`
	TargetValue int    `json:"target_value" validate:"required,min=1,max=9999"`
}

type GoalProgress struct {
	Goal         domain.StudyGoal `json:"goal"`
	CurrentValue int              `json:"current_value"`
	Completed    bool             `json:"completed"`
	Percent      float64          `json:"percent"`
}
