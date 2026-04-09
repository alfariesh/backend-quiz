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

var _ port.StatsServicer = (*StatsService)(nil)

type StatsService struct {
	reviewRepo  domain.ReviewRepository
	sessionRepo domain.StudySessionRepository
	statsRepo   domain.StatsRepository
	cardRepo    domain.CardRepository
	deckRepo    domain.DeckRepository
	quizRepo    domain.QuizRepository
	attemptRepo domain.QuizAttemptRepository
}

func NewStatsService(
	reviewRepo domain.ReviewRepository,
	sessionRepo domain.StudySessionRepository,
	statsRepo domain.StatsRepository,
	cardRepo domain.CardRepository,
	deckRepo domain.DeckRepository,
	quizRepo domain.QuizRepository,
	attemptRepo domain.QuizAttemptRepository,
) *StatsService {
	return &StatsService{
		reviewRepo:  reviewRepo,
		sessionRepo: sessionRepo,
		statsRepo:   statsRepo,
		cardRepo:    cardRepo,
		deckRepo:    deckRepo,
		quizRepo:    quizRepo,
		attemptRepo: attemptRepo,
	}
}

func (s *StatsService) Overview(ctx context.Context, userID uuid.UUID) (*dto.OverviewStats, error) {
	streak, err := s.statsRepo.GetStreak(ctx, userID)
	if err != nil {
		streak = 0
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	thirtyDaysAgo := today.AddDate(0, 0, -30)

	todayReviews, err := s.reviewRepo.CountByUserAndDate(ctx, userID, today)
	if err != nil {
		todayReviews = 0
	}

	dailyCounts, err := s.reviewRepo.GetReviewCountsPerDay(ctx, userID, thirtyDaysAgo, tomorrow)
	if err != nil {
		dailyCounts = nil
	}

	totalReviews := 0
	totalCorrect := 0
	for _, d := range dailyCounts {
		totalReviews += d.Count
		totalCorrect += d.Correct
	}

	var retention float64
	if totalReviews > 0 {
		retention = math.Round(float64(totalCorrect)/float64(totalReviews)*10000) / 100
	}

	return &dto.OverviewStats{
		TotalReviews:  totalReviews,
		Streak:        streak,
		RetentionRate: retention,
		TodayReviews:  todayReviews,
	}, nil
}

func (s *StatsService) Heatmap(ctx context.Context, userID uuid.UUID) ([]dto.HeatmapEntry, error) {
	now := time.Now()
	yearAgo := now.AddDate(-1, 0, 0)

	counts, err := s.reviewRepo.GetReviewCountsPerDay(ctx, userID, yearAgo, now)
	if err != nil {
		return nil, err
	}

	entries := make([]dto.HeatmapEntry, len(counts))
	for i, c := range counts {
		entries[i] = dto.HeatmapEntry{
			Date:  c.Date.Format("2006-01-02"),
			Count: c.Count,
		}
	}
	return entries, nil
}

func (s *StatsService) Forecast(ctx context.Context, userID uuid.UUID) ([]dto.ForecastDay, error) {
	now := time.Now()
	forecast := make([]dto.ForecastDay, 30)

	for i := range 30 {
		day := now.AddDate(0, 0, i)
		forecast[i] = dto.ForecastDay{
			Date: day.Format("2006-01-02"),
		}
	}

	return forecast, nil
}

func (s *StatsService) Leaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.statsRepo.GetGlobalLeaderboard(ctx, limit)
}

func (s *StatsService) DeckStats(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) (map[string]any, error) {
	counts, err := s.cardRepo.CountByState(ctx, deckID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	dueCount, err := s.cardRepo.CountDue(ctx, deckID, now)
	if err != nil {
		dueCount = 0
	}

	total := 0
	for _, c := range counts {
		total += c
	}

	return map[string]any{
		"total_cards":     total,
		"new_cards":       counts[domain.CardStateNew],
		"learning_cards":  counts[domain.CardStateLearning],
		"review_cards":    counts[domain.CardStateReview],
		"relearning_cards": counts[domain.CardStateRelearning],
		"due_count":       dueCount,
	}, nil
}

// Mastery Level


func masteryLevel(percent float64) string {
	switch {
	case percent >= 86:
		return "mastered"
	case percent >= 61:
		return "advanced"
	case percent >= 31:
		return "intermediate"
	default:
		return "beginner"
	}
}

func (s *StatsService) Mastery(ctx context.Context, userID uuid.UUID) ([]dto.DeckMastery, error) {
	decks, _, err := s.deckRepo.ListByUserID(ctx, userID, 1000, 0)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	var results []dto.DeckMastery
	for _, d := range decks {
		totalCards, matureCards, avgStability, err := s.cardRepo.GetDeckMasteryStats(ctx, d.ID)
		if err != nil || totalCards == 0 {
			continue
		}

		// Get retention rate from review logs for this deck's cards
		dailyCounts, err := s.reviewRepo.GetReviewCountsPerDay(ctx, userID, thirtyDaysAgo, now)
		if err != nil {
			dailyCounts = nil
		}
		totalReviews := 0
		totalCorrect := 0
		for _, dc := range dailyCounts {
			totalReviews += dc.Count
			totalCorrect += dc.Correct
		}
		var retention float64
		if totalReviews > 0 {
			retention = math.Round(float64(totalCorrect)/float64(totalReviews)*10000) / 100
		}

		maturePercent := math.Round(float64(matureCards)/float64(totalCards)*10000) / 100

		// Mastery = weighted: 40% mature%, 40% retention, 20% stability score
		stabilityScore := math.Min(avgStability/30.0*100, 100)
		mastery := math.Round(maturePercent*0.4+retention*0.4+stabilityScore*0.2*100) / 100
		if mastery > 100 {
			mastery = 100
		}

		results = append(results, dto.DeckMastery{
			DeckID:         d.ID,
			DeckName:       d.Name,
			MasteryPercent: mastery,
			MasteryLevel:   masteryLevel(mastery),
			RetentionRate:  retention,
			MaturePercent:  maturePercent,
			AvgStability:   math.Round(avgStability*100) / 100,
			TotalCards:     totalCards,
			MatureCards:    matureCards,
		})
	}

	return results, nil
}

// Weak Area Detection

func (s *StatsService) WeakAreas(ctx context.Context, userID uuid.UUID) (*dto.WeakAreasResponse, error) {
	cards, err := s.cardRepo.GetWeakCards(ctx, userID, 50)
	if err != nil {
		return nil, err
	}

	// Aggregate by tag
	tagStats := make(map[string]struct {
		count         int
		totalLapses   int
		totalStability float64
	})

	for _, c := range cards {
		for _, tag := range c.Tags {
			s := tagStats[tag]
			s.count++
			s.totalLapses += c.Lapses
			s.totalStability += c.Stability
			tagStats[tag] = s
		}
	}

	var areas []dto.WeakArea
	for tag, stats := range tagStats {
		if stats.count < 2 {
			continue
		}
		areas = append(areas, dto.WeakArea{
			Tag:          tag,
			WeakCards:    stats.count,
			AvgLapses:    math.Round(float64(stats.totalLapses)/float64(stats.count)*100) / 100,
			AvgStability: math.Round(stats.totalStability/float64(stats.count)*100) / 100,
		})
	}

	// Return top 20 weak cards
	topCards := cards
	if len(topCards) > 20 {
		topCards = topCards[:20]
	}

	return &dto.WeakAreasResponse{
		WeakAreas: areas,
		WeakCards: topCards,
	}, nil
}

// Pre-test / Post-test Comparison

func (s *StatsService) TestComparison(ctx context.Context, userID, deckID uuid.UUID) (*dto.TestComparison, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	result := &dto.TestComparison{
		DeckID:   deckID,
		DeckName: deck.Name,
	}

	// Find pretest quizzes
	preQuizzes, err := s.quizRepo.ListByDeckAndType(ctx, deckID, domain.QuizTypePretest)
	if err == nil && len(preQuizzes) > 0 {
		tr := s.buildTestResult(ctx, preQuizzes)
		if tr != nil {
			result.PreTest = tr
		}
	}

	// Find posttest quizzes
	postQuizzes, err := s.quizRepo.ListByDeckAndType(ctx, deckID, domain.QuizTypePosttest)
	if err == nil && len(postQuizzes) > 0 {
		tr := s.buildTestResult(ctx, postQuizzes)
		if tr != nil {
			result.PostTest = tr
		}
	}

	// Calculate improvement
	if result.PreTest != nil && result.PostTest != nil {
		improvement := result.PostTest.ScorePercent - result.PreTest.ScorePercent
		improvement = math.Round(improvement*100) / 100
		result.Improvement = &improvement
	}

	return result, nil
}

func (s *StatsService) buildTestResult(ctx context.Context, quizzes []domain.Quiz) *dto.TestResult {
	var best *domain.QuizAttempt

	for _, q := range quizzes {
		attempt, err := s.attemptRepo.GetBestAttemptByQuizID(ctx, q.ID)
		if err != nil {
			continue
		}
		if best == nil || attempt.Score > best.Score {
			best = attempt
		}
	}

	if best == nil {
		return nil
	}

	var scorePercent float64
	if best.TotalPoints > 0 {
		scorePercent = math.Round(float64(best.Score)/float64(best.TotalPoints)*10000) / 100
	}

	// Count total attempts across all quizzes
	totalAttempts := 0
	for _, q := range quizzes {
		attempts, _, err := s.attemptRepo.ListByQuizID(ctx, q.ID, 1000, 0)
		if err == nil {
			totalAttempts += len(attempts)
		}
	}

	return &dto.TestResult{
		QuizID:       best.QuizID,
		BestScore:    best.Score,
		TotalPoints:  best.TotalPoints,
		ScorePercent: scorePercent,
		AttemptCount: totalAttempts,
		LastAttempt:  best.StartedAt,
	}
}
