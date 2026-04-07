package service

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type StatsService struct {
	reviewRepo  domain.ReviewRepository
	sessionRepo domain.StudySessionRepository
	statsRepo   domain.StatsRepository
	cardRepo    domain.CardRepository
}

func NewStatsService(
	reviewRepo domain.ReviewRepository,
	sessionRepo domain.StudySessionRepository,
	statsRepo domain.StatsRepository,
	cardRepo domain.CardRepository,
) *StatsService {
	return &StatsService{
		reviewRepo:  reviewRepo,
		sessionRepo: sessionRepo,
		statsRepo:   statsRepo,
		cardRepo:    cardRepo,
	}
}

type OverviewStats struct {
	TotalReviews   int     `json:"total_reviews"`
	Streak         int     `json:"streak"`
	RetentionRate  float64 `json:"retention_rate"`
	TodayReviews   int     `json:"today_reviews"`
	TodayNewCards  int     `json:"today_new_cards"`
}

type HeatmapEntry struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type ForecastDay struct {
	Date    string `json:"date"`
	DueNew  int    `json:"due_new"`
	DueReview int  `json:"due_review"`
}

func (s *StatsService) Overview(ctx context.Context, userID uuid.UUID) (*OverviewStats, error) {
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

	return &OverviewStats{
		TotalReviews:  totalReviews,
		Streak:        streak,
		RetentionRate: retention,
		TodayReviews:  todayReviews,
	}, nil
}

func (s *StatsService) Heatmap(ctx context.Context, userID uuid.UUID) ([]HeatmapEntry, error) {
	now := time.Now()
	yearAgo := now.AddDate(-1, 0, 0)

	counts, err := s.reviewRepo.GetReviewCountsPerDay(ctx, userID, yearAgo, now)
	if err != nil {
		return nil, err
	}

	entries := make([]HeatmapEntry, len(counts))
	for i, c := range counts {
		entries[i] = HeatmapEntry{
			Date:  c.Date.Format("2006-01-02"),
			Count: c.Count,
		}
	}
	return entries, nil
}

func (s *StatsService) Forecast(ctx context.Context, userID uuid.UUID) ([]ForecastDay, error) {
	now := time.Now()
	forecast := make([]ForecastDay, 30)

	for i := range 30 {
		day := now.AddDate(0, 0, i)
		forecast[i] = ForecastDay{
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
