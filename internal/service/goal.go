package service

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	"github.com/rekanesiads/backend-quiz/internal/port"
)

var _ port.GoalServicer = (*GoalService)(nil)

type GoalService struct {
	goalRepo   domain.GoalRepository
	reviewRepo domain.ReviewRepository
}

func NewGoalService(goalRepo domain.GoalRepository, reviewRepo domain.ReviewRepository) *GoalService {
	return &GoalService{goalRepo: goalRepo, reviewRepo: reviewRepo}
}

type SetGoalRequest = dto.SetGoalRequest
type GoalProgress = dto.GoalProgress

func (s *GoalService) SetGoal(ctx context.Context, userID uuid.UUID, req SetGoalRequest) (*domain.StudyGoal, error) {
	goal := &domain.StudyGoal{
		UserID:      userID,
		GoalType:    req.GoalType,
		TargetValue: req.TargetValue,
		IsActive:    true,
	}
	if err := s.goalRepo.Upsert(ctx, goal); err != nil {
		return nil, err
	}
	return goal, nil
}

func (s *GoalService) ListWithProgress(ctx context.Context, userID uuid.UUID) ([]GoalProgress, error) {
	goals, err := s.goalRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)

	var results []GoalProgress
	for _, goal := range goals {
		current := 0

		switch goal.GoalType {
		case domain.GoalTypeDailyReviews:
			count, err := s.reviewRepo.CountByUserAndDate(ctx, userID, today)
			if err == nil {
				current = count
			}

		case domain.GoalTypeDailyNew:
			// Count today's reviews where state was New (rating on new cards)
			reviews, _, err := s.reviewRepo.ListByUserID(ctx, userID, today, tomorrow, 10000, 0)
			if err == nil {
				for _, r := range reviews {
					if r.State == domain.CardStateNew {
						current++
					}
				}
			}

		case domain.GoalTypeWeeklyReviews:
			counts, err := s.reviewRepo.GetReviewCountsPerDay(ctx, userID, weekAgo, tomorrow)
			if err == nil {
				for _, c := range counts {
					current += c.Count
				}
			}

		case domain.GoalTypeDailyMinutes:
			reviews, _, err := s.reviewRepo.ListByUserID(ctx, userID, today, tomorrow, 10000, 0)
			if err == nil {
				totalMS := 0
				for _, r := range reviews {
					totalMS += r.DurationMS
				}
				current = totalMS / 60000 // convert to minutes
			}
		}

		percent := 0.0
		if goal.TargetValue > 0 {
			percent = math.Min(math.Round(float64(current)/float64(goal.TargetValue)*10000)/100, 100)
		}

		results = append(results, GoalProgress{
			Goal:         goal,
			CurrentValue: current,
			Completed:    current >= goal.TargetValue,
			Percent:      percent,
		})
	}

	return results, nil
}

func (s *GoalService) DeleteGoal(ctx context.Context, userID, goalID uuid.UUID) error {
	goal, err := s.goalRepo.GetByID(ctx, goalID)
	if err != nil {
		return err
	}
	if goal.UserID != userID {
		return domain.ErrForbidden
	}
	return s.goalRepo.Delete(ctx, goalID)
}
