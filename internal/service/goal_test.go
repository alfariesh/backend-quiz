package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/dto"
	mockdomain "github.com/alfariesh/backend-quiz/internal/mocks/domain"
)

// --- SetGoal ---

func TestGoalService_SetGoal_Success(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	goalRepo.On("Upsert", ctx, mock.AnythingOfType("*domain.StudyGoal")).Return(nil)

	goal, err := svc.SetGoal(ctx, userID, dto.SetGoalRequest{
		GoalType:    domain.GoalTypeDailyReviews,
		TargetValue: 50,
	})

	require.NoError(t, err)
	assert.Equal(t, domain.GoalTypeDailyReviews, goal.GoalType)
	assert.Equal(t, 50, goal.TargetValue)
	assert.True(t, goal.IsActive)
	assert.Equal(t, userID, goal.UserID)
}

func TestGoalService_SetGoal_UpsertError(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()

	goalRepo.On("Upsert", ctx, mock.AnythingOfType("*domain.StudyGoal")).Return(assert.AnError)

	_, err := svc.SetGoal(ctx, uuid.New(), dto.SetGoalRequest{
		GoalType:    domain.GoalTypeDailyReviews,
		TargetValue: 50,
	})
	assert.Error(t, err)
}

func TestGoalService_ListWithProgress_RepoError(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	goalRepo.On("ListByUserID", ctx, userID).Return(nil, assert.AnError)

	_, err := svc.ListWithProgress(ctx, userID)
	assert.Error(t, err)
}

// --- DeleteGoal ---

func TestGoalService_DeleteGoal_Success(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	goalID := uuid.New()
	goal := &domain.StudyGoal{ID: goalID, UserID: userID}

	goalRepo.On("GetByID", ctx, goalID).Return(goal, nil)
	goalRepo.On("Delete", ctx, goalID).Return(nil)

	err := svc.DeleteGoal(ctx, userID, goalID)
	assert.NoError(t, err)
}

func TestGoalService_DeleteGoal_Forbidden(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()

	goalID := uuid.New()
	goal := &domain.StudyGoal{ID: goalID, UserID: uuid.New()}

	goalRepo.On("GetByID", ctx, goalID).Return(goal, nil)

	err := svc.DeleteGoal(ctx, uuid.New(), goalID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestGoalService_DeleteGoal_NotFound(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()

	goalID := uuid.New()
	goalRepo.On("GetByID", ctx, goalID).Return(nil, domain.ErrNotFound)

	err := svc.DeleteGoal(ctx, uuid.New(), goalID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- ListWithProgress ---

func TestGoalService_ListWithProgress_DailyReviews(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	goals := []domain.StudyGoal{
		{ID: uuid.New(), UserID: userID, GoalType: domain.GoalTypeDailyReviews, TargetValue: 20, IsActive: true},
	}
	goalRepo.On("ListByUserID", ctx, userID).Return(goals, nil)
	reviewRepo.On("CountByUserAndDate", ctx, userID, mock.AnythingOfType("time.Time")).Return(15, nil)

	progress, err := svc.ListWithProgress(ctx, userID)

	require.NoError(t, err)
	require.Len(t, progress, 1)
	assert.Equal(t, 15, progress[0].CurrentValue)
	assert.False(t, progress[0].Completed)
	assert.Equal(t, 75.0, progress[0].Percent)
}

func TestGoalService_ListWithProgress_DailyReviews_Completed(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	goals := []domain.StudyGoal{
		{ID: uuid.New(), UserID: userID, GoalType: domain.GoalTypeDailyReviews, TargetValue: 10, IsActive: true},
	}
	goalRepo.On("ListByUserID", ctx, userID).Return(goals, nil)
	reviewRepo.On("CountByUserAndDate", ctx, userID, mock.AnythingOfType("time.Time")).Return(25, nil)

	progress, err := svc.ListWithProgress(ctx, userID)

	require.NoError(t, err)
	assert.True(t, progress[0].Completed)
	assert.Equal(t, 100.0, progress[0].Percent) // capped at 100
}

func TestGoalService_ListWithProgress_DailyNew(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	goals := []domain.StudyGoal{
		{ID: uuid.New(), UserID: userID, GoalType: domain.GoalTypeDailyNew, TargetValue: 10, IsActive: true},
	}
	goalRepo.On("ListByUserID", ctx, userID).Return(goals, nil)

	// Return reviews where some are new cards
	reviews := []domain.ReviewLog{
		{State: domain.CardStateNew},
		{State: domain.CardStateNew},
		{State: domain.CardStateNew},
		{State: domain.CardStateReview},
		{State: domain.CardStateLearning},
	}
	reviewRepo.On("ListByUserID", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10000, 0).Return(reviews, 5, nil)

	progress, err := svc.ListWithProgress(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, 3, progress[0].CurrentValue) // only new cards
	assert.Equal(t, 30.0, progress[0].Percent)
}

func TestGoalService_ListWithProgress_WeeklyReviews(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	goals := []domain.StudyGoal{
		{ID: uuid.New(), UserID: userID, GoalType: domain.GoalTypeWeeklyReviews, TargetValue: 100, IsActive: true},
	}
	goalRepo.On("ListByUserID", ctx, userID).Return(goals, nil)

	counts := []domain.DailyReviewCount{
		{Date: time.Now(), Count: 20, Correct: 18},
		{Date: time.Now().AddDate(0, 0, -1), Count: 15, Correct: 12},
		{Date: time.Now().AddDate(0, 0, -2), Count: 25, Correct: 20},
	}
	reviewRepo.On("GetReviewCountsPerDay", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(counts, nil)

	progress, err := svc.ListWithProgress(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, 60, progress[0].CurrentValue) // 20+15+25
	assert.Equal(t, 60.0, progress[0].Percent)
}

func TestGoalService_ListWithProgress_DailyMinutes(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	goals := []domain.StudyGoal{
		{ID: uuid.New(), UserID: userID, GoalType: domain.GoalTypeDailyMinutes, TargetValue: 30, IsActive: true},
	}
	goalRepo.On("ListByUserID", ctx, userID).Return(goals, nil)

	reviews := []domain.ReviewLog{
		{DurationMS: 600000},  // 10 min
		{DurationMS: 300000},  // 5 min
		{DurationMS: 900000},  // 15 min
	}
	reviewRepo.On("ListByUserID", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10000, 0).Return(reviews, 3, nil)

	progress, err := svc.ListWithProgress(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, 30, progress[0].CurrentValue) // 30 minutes
	assert.True(t, progress[0].Completed)
}

func TestGoalService_ListWithProgress_Empty(t *testing.T) {
	goalRepo := mockdomain.NewMockGoalRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewGoalService(goalRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	goalRepo.On("ListByUserID", ctx, userID).Return([]domain.StudyGoal{}, nil)

	progress, err := svc.ListWithProgress(ctx, userID)

	require.NoError(t, err)
	assert.Nil(t, progress) // no goals
}
