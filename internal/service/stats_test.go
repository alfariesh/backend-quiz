package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	mockdomain "github.com/rekanesiads/backend-quiz/internal/mocks/domain"
)


func TestMasteryLevel(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		level   string
	}{
		// Beginner: < 31
		{"zero", 0, "beginner"},
		{"low beginner", 10, "beginner"},
		{"upper beginner", 30, "beginner"},
		{"beginner boundary", 30.99, "beginner"},

		// Intermediate: 31-60
		{"intermediate start", 31, "intermediate"},
		{"mid intermediate", 45, "intermediate"},
		{"upper intermediate", 60, "intermediate"},
		{"intermediate boundary", 60.99, "intermediate"},

		// Advanced: 61-85
		{"advanced start", 61, "advanced"},
		{"mid advanced", 75, "advanced"},
		{"upper advanced", 85, "advanced"},
		{"advanced boundary", 85.99, "advanced"},

		// Mastered: >= 86
		{"mastered start", 86, "mastered"},
		{"high mastered", 95, "mastered"},
		{"perfect", 100, "mastered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.level, masteryLevel(tt.percent))
		})
	}
}

// ============================================================
// StatsService Method Tests
// ============================================================

// --- Overview ---

func TestStatsService_Overview_Success(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	statsRepo.On("GetStreak", ctx, userID).Return(5, nil)
	reviewRepo.On("CountByUserAndDate", ctx, userID, mock.AnythingOfType("time.Time")).Return(12, nil)
	reviewRepo.On("GetReviewCountsPerDay", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]domain.DailyReviewCount{
		{Date: time.Now(), Count: 20, Correct: 18},
		{Date: time.Now().AddDate(0, 0, -1), Count: 30, Correct: 24},
	}, nil)

	overview, err := svc.Overview(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, 5, overview.Streak)
	assert.Equal(t, 12, overview.TodayReviews)
	assert.Equal(t, 50, overview.TotalReviews) // 20+30
	assert.Equal(t, 84.0, overview.RetentionRate) // 42/50 = 0.84
}

func TestStatsService_Overview_NoData(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	statsRepo.On("GetStreak", ctx, userID).Return(0, assert.AnError)
	reviewRepo.On("CountByUserAndDate", ctx, userID, mock.AnythingOfType("time.Time")).Return(0, assert.AnError)
	reviewRepo.On("GetReviewCountsPerDay", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]domain.DailyReviewCount(nil), assert.AnError)

	overview, err := svc.Overview(ctx, userID)

	require.NoError(t, err) // graceful — no hard errors
	assert.Equal(t, 0, overview.Streak)
	assert.Equal(t, 0, overview.TodayReviews)
	assert.Equal(t, 0, overview.TotalReviews)
	assert.Equal(t, float64(0), overview.RetentionRate)
}

// --- Heatmap ---

func TestStatsService_Heatmap_Success(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	counts := []domain.DailyReviewCount{
		{Date: now, Count: 10},
		{Date: now.AddDate(0, 0, -1), Count: 5},
	}
	reviewRepo.On("GetReviewCountsPerDay", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(counts, nil)

	entries, err := svc.Heatmap(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, now.Format("2006-01-02"), entries[0].Date)
	assert.Equal(t, 10, entries[0].Count)
}

func TestStatsService_Heatmap_Error(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	reviewRepo.On("GetReviewCountsPerDay", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]domain.DailyReviewCount(nil), assert.AnError)

	_, err := svc.Heatmap(ctx, userID)
	assert.Error(t, err)
}

// --- Forecast ---

func TestStatsService_Forecast_Returns30Days(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	forecast, err := svc.Forecast(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, forecast, 30)
	assert.Equal(t, time.Now().Format("2006-01-02"), forecast[0].Date)
}

// --- Leaderboard ---

func TestStatsService_Leaderboard_Success(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	entries := []domain.LeaderboardEntry{
		{UserID: uuid.New(), DisplayName: "User1", Streak: 10, Rank: 1},
		{UserID: uuid.New(), DisplayName: "User2", Streak: 7, Rank: 2},
	}
	statsRepo.On("GetGlobalLeaderboard", ctx, 10).Return(entries, nil)

	result, err := svc.Leaderboard(ctx, 10)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "User1", result[0].DisplayName)
}

func TestStatsService_Leaderboard_DefaultLimit(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	statsRepo.On("GetGlobalLeaderboard", ctx, 20).Return([]domain.LeaderboardEntry{}, nil)

	result, err := svc.Leaderboard(ctx, 0) // invalid → defaults to 20

	require.NoError(t, err)
	assert.Empty(t, result)
}

// --- DeckStats ---

func TestStatsService_DeckStats_Success(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("CountByState", ctx, deckID).Return(map[domain.CardState]int{
		domain.CardStateNew:        10,
		domain.CardStateLearning:   5,
		domain.CardStateReview:     20,
		domain.CardStateRelearning: 2,
	}, nil)
	cardRepo.On("CountDue", ctx, deckID, mock.AnythingOfType("time.Time")).Return(8, nil)

	stats, err := svc.DeckStats(ctx, userID, deckID)

	require.NoError(t, err)
	assert.Equal(t, 37, stats["total_cards"])
	assert.Equal(t, 10, stats["new_cards"])
	assert.Equal(t, 20, stats["review_cards"])
	assert.Equal(t, 8, stats["due_count"])
}

func TestStatsService_DeckStats_CountByStateError(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	deckID := uuid.New()
	cardRepo.On("CountByState", ctx, deckID).Return(map[domain.CardState]int(nil), assert.AnError)

	_, err := svc.DeckStats(ctx, uuid.New(), deckID)
	assert.Error(t, err)
}

// --- Mastery ---

func TestStatsService_Mastery_Success(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()

	deckRepo.On("ListByUserID", ctx, userID, 1000, 0).Return([]domain.DeckWithCounts{
		{Deck: domain.Deck{ID: deckID, Name: "Fiqh"}},
	}, 1, nil)

	// totalCards=100, matureCards=80, avgStability=15.0
	cardRepo.On("GetDeckMasteryStats", ctx, deckID).Return(100, 80, 15.0, nil)
	reviewRepo.On("GetReviewCountsPerDay", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]domain.DailyReviewCount{
		{Count: 50, Correct: 45},
	}, nil)

	results, err := svc.Mastery(ctx, userID)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Fiqh", results[0].DeckName)
	assert.Equal(t, 100, results[0].TotalCards)
	assert.Equal(t, 80, results[0].MatureCards)
	assert.Equal(t, 80.0, results[0].MaturePercent)
	assert.Greater(t, results[0].MasteryPercent, float64(0))
	assert.NotEmpty(t, results[0].MasteryLevel)
}

func TestStatsService_Mastery_SkipsEmptyDecks(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()

	deckRepo.On("ListByUserID", ctx, userID, 1000, 0).Return([]domain.DeckWithCounts{
		{Deck: domain.Deck{ID: deckID, Name: "Empty"}},
	}, 1, nil)
	cardRepo.On("GetDeckMasteryStats", ctx, deckID).Return(0, 0, 0.0, nil) // totalCards=0

	results, err := svc.Mastery(ctx, userID)

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestStatsService_Mastery_NoDecks(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckRepo.On("ListByUserID", ctx, userID, 1000, 0).Return([]domain.DeckWithCounts{}, 0, nil)

	results, err := svc.Mastery(ctx, userID)

	require.NoError(t, err)
	assert.Nil(t, results)
}

// --- WeakAreas ---

func TestStatsService_WeakAreas_Success(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	weakCards := []domain.Card{
		{ID: uuid.New(), Tags: []string{"topic:fiqh", "chapter:salat"}, Lapses: 5, Stability: 1.2},
		{ID: uuid.New(), Tags: []string{"topic:fiqh", "chapter:zakat"}, Lapses: 3, Stability: 2.0},
		{ID: uuid.New(), Tags: []string{"topic:fiqh", "chapter:salat"}, Lapses: 4, Stability: 1.5},
	}
	cardRepo.On("GetWeakCards", ctx, userID, 50).Return(weakCards, nil)

	resp, err := svc.WeakAreas(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, resp.WeakCards, 3)

	// topic:fiqh has 3 cards (>=2), chapter:salat has 2 cards (>=2), chapter:zakat has 1 (filtered out)
	foundFiqh := false
	foundSalat := false
	for _, area := range resp.WeakAreas {
		if area.Tag == "topic:fiqh" {
			foundFiqh = true
			assert.Equal(t, 3, area.WeakCards)
			assert.Equal(t, 4.0, area.AvgLapses) // (5+3+4)/3 = 4.0
		}
		if area.Tag == "chapter:salat" {
			foundSalat = true
			assert.Equal(t, 2, area.WeakCards)
		}
	}
	assert.True(t, foundFiqh)
	assert.True(t, foundSalat)
}

func TestStatsService_WeakAreas_Empty(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	cardRepo.On("GetWeakCards", ctx, userID, 50).Return([]domain.Card{}, nil)

	resp, err := svc.WeakAreas(ctx, userID)

	require.NoError(t, err)
	assert.Empty(t, resp.WeakCards)
	assert.Nil(t, resp.WeakAreas)
}

func TestStatsService_WeakAreas_CapsAt20Cards(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	// 25 weak cards
	cards := make([]domain.Card, 25)
	for i := range cards {
		cards[i] = domain.Card{ID: uuid.New(), Tags: []string{"topic:test"}, Lapses: 3, Stability: 1.0}
	}
	cardRepo.On("GetWeakCards", ctx, userID, 50).Return(cards, nil)

	resp, err := svc.WeakAreas(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, resp.WeakCards, 20) // capped at 20
}

func TestStatsService_WeakAreas_Error(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()
	userID := uuid.New()

	cardRepo.On("GetWeakCards", ctx, userID, 50).Return([]domain.Card(nil), assert.AnError)

	_, err := svc.WeakAreas(ctx, userID)
	assert.Error(t, err)
}

// --- TestComparison ---

func TestStatsService_TestComparison_WithImprovement(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	preQuizID := uuid.New()
	postQuizID := uuid.New()

	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID, Name: "Fiqh"}, nil)

	quizRepo.On("ListByDeckAndType", ctx, deckID, domain.QuizTypePretest).Return([]domain.Quiz{
		{ID: preQuizID},
	}, nil)
	quizRepo.On("ListByDeckAndType", ctx, deckID, domain.QuizTypePosttest).Return([]domain.Quiz{
		{ID: postQuizID},
	}, nil)

	// Pre-test: score 6/10 = 60%
	attemptRepo.On("GetBestAttemptByQuizID", ctx, preQuizID).Return(&domain.QuizAttempt{
		ID: uuid.New(), QuizID: preQuizID, Score: 6, TotalPoints: 10, StartedAt: time.Now().AddDate(0, 0, -7),
	}, nil)
	attemptRepo.On("ListByQuizID", ctx, preQuizID, 1000, 0).Return([]domain.QuizAttempt{{}, {}}, 2, nil)

	// Post-test: score 9/10 = 90%
	attemptRepo.On("GetBestAttemptByQuizID", ctx, postQuizID).Return(&domain.QuizAttempt{
		ID: uuid.New(), QuizID: postQuizID, Score: 9, TotalPoints: 10, StartedAt: time.Now(),
	}, nil)
	attemptRepo.On("ListByQuizID", ctx, postQuizID, 1000, 0).Return([]domain.QuizAttempt{{}}, 1, nil)

	result, err := svc.TestComparison(ctx, userID, deckID)

	require.NoError(t, err)
	assert.Equal(t, "Fiqh", result.DeckName)
	require.NotNil(t, result.PreTest)
	require.NotNil(t, result.PostTest)
	assert.Equal(t, 60.0, result.PreTest.ScorePercent)
	assert.Equal(t, 90.0, result.PostTest.ScorePercent)
	require.NotNil(t, result.Improvement)
	assert.Equal(t, 30.0, *result.Improvement)
	assert.Equal(t, 2, result.PreTest.AttemptCount)
}

func TestStatsService_TestComparison_NoTests(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()

	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID, Name: "Empty"}, nil)
	quizRepo.On("ListByDeckAndType", ctx, deckID, domain.QuizTypePretest).Return([]domain.Quiz{}, nil)
	quizRepo.On("ListByDeckAndType", ctx, deckID, domain.QuizTypePosttest).Return([]domain.Quiz{}, nil)

	result, err := svc.TestComparison(ctx, userID, deckID)

	require.NoError(t, err)
	assert.Nil(t, result.PreTest)
	assert.Nil(t, result.PostTest)
	assert.Nil(t, result.Improvement)
}

func TestStatsService_TestComparison_Forbidden(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	deckID := uuid.New()
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: uuid.New()}, nil)

	_, err := svc.TestComparison(ctx, uuid.New(), deckID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestStatsService_TestComparison_OnlyPretest(t *testing.T) {
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	svc := NewStatsService(reviewRepo, mockdomain.NewMockStudySessionRepository(t), statsRepo, cardRepo, deckRepo, quizRepo, attemptRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	preQuizID := uuid.New()

	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	quizRepo.On("ListByDeckAndType", ctx, deckID, domain.QuizTypePretest).Return([]domain.Quiz{
		{ID: preQuizID},
	}, nil)
	quizRepo.On("ListByDeckAndType", ctx, deckID, domain.QuizTypePosttest).Return([]domain.Quiz{}, nil)

	attemptRepo.On("GetBestAttemptByQuizID", ctx, preQuizID).Return(&domain.QuizAttempt{
		QuizID: preQuizID, Score: 5, TotalPoints: 10, StartedAt: time.Now(),
	}, nil)
	attemptRepo.On("ListByQuizID", ctx, preQuizID, 1000, 0).Return([]domain.QuizAttempt{{}}, 1, nil)

	result, err := svc.TestComparison(ctx, userID, deckID)

	require.NoError(t, err)
	assert.NotNil(t, result.PreTest)
	assert.Nil(t, result.PostTest)
	assert.Nil(t, result.Improvement) // no post-test → no improvement calc
}
