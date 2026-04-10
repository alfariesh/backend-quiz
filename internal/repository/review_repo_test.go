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

func TestReviewRepo_CreateAndListByCardID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	reviewRepo := NewReviewRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "review@example.com")
	deck := createTestDeck(t, pool, user.ID, "Review Deck")
	card := createTestCard(t, pool, deck.ID, "Q", "A")

	log := &domain.ReviewLog{
		CardID:     card.ID,
		UserID:     user.ID,
		Rating:     domain.RatingGood,
		State:      domain.CardStateNew,
		Stability:  1.0,
		Difficulty: 5.0,
		DurationMS: 3000,
		Source:     domain.ReviewSourceFlashcard,
		ReviewedAt: time.Now().UTC(),
	}
	err := reviewRepo.Create(ctx, log)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, log.ID)

	logs, err := reviewRepo.ListByCardID(ctx, card.ID)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, domain.RatingGood, logs[0].Rating)
	assert.Equal(t, domain.ReviewSourceFlashcard, logs[0].Source)
}

func TestReviewRepo_ListByUserID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	reviewRepo := NewReviewRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "reviewlist@example.com")
	deck := createTestDeck(t, pool, user.ID, "Review List Deck")
	card := createTestCard(t, pool, deck.ID, "Q", "A")

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		require.NoError(t, reviewRepo.Create(ctx, &domain.ReviewLog{
			CardID:     card.ID,
			UserID:     user.ID,
			Rating:     domain.RatingGood,
			State:      domain.CardStateNew,
			Stability:  1.0,
			Difficulty: 5.0,
			DurationMS: 1000,
			ReviewedAt: now,
		}))
	}

	logs, total, err := reviewRepo.ListByUserID(ctx, user.ID, now.Add(-time.Hour), now.Add(time.Hour), 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, logs, 3)
}

func TestReviewRepo_CountByUserAndDate(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	reviewRepo := NewReviewRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "reviewcount@example.com")
	deck := createTestDeck(t, pool, user.ID, "Count Deck")
	card := createTestCard(t, pool, deck.ID, "Q", "A")

	now := time.Now().UTC()
	for i := 0; i < 2; i++ {
		require.NoError(t, reviewRepo.Create(ctx, &domain.ReviewLog{
			CardID:     card.ID,
			UserID:     user.ID,
			Rating:     domain.RatingGood,
			State:      domain.CardStateNew,
			Stability:  1.0,
			Difficulty: 5.0,
			DurationMS: 500,
			ReviewedAt: now,
		}))
	}

	count, err := reviewRepo.CountByUserAndDate(ctx, user.ID, now)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestReviewRepo_GetReviewCountsPerDay(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	reviewRepo := NewReviewRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "reviewperday@example.com")
	deck := createTestDeck(t, pool, user.ID, "PerDay Deck")
	card := createTestCard(t, pool, deck.ID, "Q", "A")

	now := time.Now().UTC()
	// Create reviews with rating Good (3) — counts as correct
	require.NoError(t, reviewRepo.Create(ctx, &domain.ReviewLog{
		CardID: card.ID, UserID: user.ID, Rating: domain.RatingGood,
		State: domain.CardStateNew, Stability: 1.0, Difficulty: 5.0,
		DurationMS: 500, ReviewedAt: now,
	}))
	// Rating Again (1) — counts as incorrect
	require.NoError(t, reviewRepo.Create(ctx, &domain.ReviewLog{
		CardID: card.ID, UserID: user.ID, Rating: domain.RatingAgain,
		State: domain.CardStateNew, Stability: 1.0, Difficulty: 5.0,
		DurationMS: 500, ReviewedAt: now,
	}))

	counts, err := reviewRepo.GetReviewCountsPerDay(ctx, user.ID, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	assert.NotEmpty(t, counts)
	assert.Equal(t, 2, counts[0].Count)
}
