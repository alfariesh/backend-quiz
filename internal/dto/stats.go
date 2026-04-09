package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type OverviewStats struct {
	TotalReviews  int     `json:"total_reviews"`
	Streak        int     `json:"streak"`
	RetentionRate float64 `json:"retention_rate"`
	TodayReviews  int     `json:"today_reviews"`
	TodayNewCards int     `json:"today_new_cards"`
}

type HeatmapEntry struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type ForecastDay struct {
	Date      string `json:"date"`
	DueNew    int    `json:"due_new"`
	DueReview int    `json:"due_review"`
}

type DeckMastery struct {
	DeckID         uuid.UUID `json:"deck_id"`
	DeckName       string    `json:"deck_name"`
	MasteryPercent float64   `json:"mastery_percent"`
	MasteryLevel   string    `json:"mastery_level"`
	RetentionRate  float64   `json:"retention_rate"`
	MaturePercent  float64   `json:"mature_percent"`
	AvgStability   float64   `json:"avg_stability"`
	TotalCards     int       `json:"total_cards"`
	MatureCards    int       `json:"mature_cards"`
}

type WeakArea struct {
	Tag          string  `json:"tag"`
	WeakCards    int     `json:"weak_cards"`
	AvgLapses    float64 `json:"avg_lapses"`
	AvgStability float64 `json:"avg_stability"`
}

type WeakAreasResponse struct {
	WeakAreas []WeakArea    `json:"weak_areas"`
	WeakCards []domain.Card `json:"weak_cards"`
}

type TestResult struct {
	QuizID       uuid.UUID `json:"quiz_id"`
	BestScore    int       `json:"best_score"`
	TotalPoints  int       `json:"total_points"`
	ScorePercent float64   `json:"score_percent"`
	AttemptCount int       `json:"attempt_count"`
	LastAttempt  time.Time `json:"last_attempt"`
}

type TestComparison struct {
	DeckID      uuid.UUID   `json:"deck_id"`
	DeckName    string      `json:"deck_name"`
	PreTest     *TestResult `json:"pre_test,omitempty"`
	PostTest    *TestResult `json:"post_test,omitempty"`
	Improvement *float64    `json:"improvement,omitempty"`
}
