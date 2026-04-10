package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

func TestStatsRepo_UpsertAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	statsRepo := NewStatsRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "stats@example.com")

	today := time.Now().UTC().Truncate(24 * time.Hour)
	retention := 0.85

	stats := &domain.DailyStats{
		UserID:          user.ID,
		Date:            today,
		NewCards:        5,
		Reviews:         20,
		Relearns:        2,
		TotalDurationMS: 30000,
		RetentionRate:   &retention,
	}
	err := statsRepo.UpsertDailyStats(ctx, stats)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, stats.ID)

	// Upsert again should update, not create duplicate
	stats.Reviews = 25
	err = statsRepo.UpsertDailyStats(ctx, stats)
	require.NoError(t, err)

	got, err := statsRepo.GetDailyStats(ctx, user.ID, today.Add(-time.Hour), today.Add(25*time.Hour))
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, 25, got[0].Reviews)
	assert.NotNil(t, got[0].RetentionRate)
	assert.InDelta(t, 0.85, *got[0].RetentionRate, 0.01)
}

func TestStatsRepo_GetDailyStats_Empty(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	statsRepo := NewStatsRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "statsempty@example.com")

	today := time.Now().UTC().Truncate(24 * time.Hour)
	got, err := statsRepo.GetDailyStats(ctx, user.ID, today, today.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestStatsRepo_GetStreak(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	statsRepo := NewStatsRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "streak@example.com")

	// No stats yet — streak should be 0
	streak, err := statsRepo.GetStreak(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, streak)

	// Add stats for today and yesterday
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 0; i < 3; i++ {
		require.NoError(t, statsRepo.UpsertDailyStats(ctx, &domain.DailyStats{
			UserID:  user.ID,
			Date:    today.AddDate(0, 0, -i),
			Reviews: 10,
		}))
	}

	streak, err = statsRepo.GetStreak(ctx, user.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, streak, 1)
}

func TestStatsRepo_GetGlobalLeaderboard(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	statsRepo := NewStatsRepository(pool)
	ctx := context.Background()

	// Create users with stats
	for i, email := range []string{"lb1@example.com", "lb2@example.com"} {
		user := createTestUser(t, userRepo, email)
		today := time.Now().UTC().Truncate(24 * time.Hour)
		require.NoError(t, statsRepo.UpsertDailyStats(ctx, &domain.DailyStats{
			UserID:  user.ID,
			Date:    today,
			Reviews: 10 * (i + 1),
		}))
	}

	entries, err := statsRepo.GetGlobalLeaderboard(ctx, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2)
}
