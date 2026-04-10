package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

func TestGoalRepo_UpsertAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	goalRepo := NewGoalRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "goal@example.com")

	goal := &domain.StudyGoal{
		UserID:      user.ID,
		GoalType:    domain.GoalTypeDailyReviews,
		TargetValue: 50,
		IsActive:    true,
	}
	err := goalRepo.Upsert(ctx, goal)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, goal.ID)

	got, err := goalRepo.GetByID(ctx, goal.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.GoalTypeDailyReviews, got.GoalType)
	assert.Equal(t, 50, got.TargetValue)
	assert.True(t, got.IsActive)
}

func TestGoalRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	goalRepo := NewGoalRepository(pool)
	ctx := context.Background()

	_, err := goalRepo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGoalRepo_Upsert_Updates(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	goalRepo := NewGoalRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "goalupd@example.com")

	goal := &domain.StudyGoal{
		UserID:      user.ID,
		GoalType:    domain.GoalTypeDailyNew,
		TargetValue: 10,
		IsActive:    true,
	}
	require.NoError(t, goalRepo.Upsert(ctx, goal))

	// Upsert same type — should update
	goal2 := &domain.StudyGoal{
		UserID:      user.ID,
		GoalType:    domain.GoalTypeDailyNew,
		TargetValue: 20,
		IsActive:    true,
	}
	require.NoError(t, goalRepo.Upsert(ctx, goal2))

	goals, err := goalRepo.ListByUserID(ctx, user.ID)
	require.NoError(t, err)

	// Should have only 1 goal of this type (upsert)
	count := 0
	for _, g := range goals {
		if g.GoalType == domain.GoalTypeDailyNew {
			count++
			assert.Equal(t, 20, g.TargetValue)
		}
	}
	assert.Equal(t, 1, count)
}

func TestGoalRepo_ListByUserID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	goalRepo := NewGoalRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "goallist@example.com")

	for _, goalType := range []string{domain.GoalTypeDailyReviews, domain.GoalTypeDailyMinutes} {
		require.NoError(t, goalRepo.Upsert(ctx, &domain.StudyGoal{
			UserID:      user.ID,
			GoalType:    goalType,
			TargetValue: 30,
			IsActive:    true,
		}))
	}

	goals, err := goalRepo.ListByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, goals, 2)
}

func TestGoalRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	goalRepo := NewGoalRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "goaldel@example.com")

	goal := &domain.StudyGoal{
		UserID:      user.ID,
		GoalType:    domain.GoalTypeWeeklyReviews,
		TargetValue: 100,
		IsActive:    true,
	}
	require.NoError(t, goalRepo.Upsert(ctx, goal))

	err := goalRepo.Delete(ctx, goal.ID)
	require.NoError(t, err)

	_, err = goalRepo.GetByID(ctx, goal.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
