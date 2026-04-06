package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ReviewLog struct {
	ID            uuid.UUID `json:"id"`
	CardID        uuid.UUID `json:"card_id"`
	UserID        uuid.UUID `json:"user_id"`
	Rating        Rating    `json:"rating"`
	State         CardState `json:"state"`
	ScheduledDays int       `json:"scheduled_days"`
	ElapsedDays   int       `json:"elapsed_days"`
	Stability     float64   `json:"stability"`
	Difficulty    float64   `json:"difficulty"`
	DurationMS    int       `json:"duration_ms"`
	ReviewedAt    time.Time `json:"reviewed_at"`
}

type ReviewRepository interface {
	Create(ctx context.Context, log *ReviewLog) error
	ListByCardID(ctx context.Context, cardID uuid.UUID) ([]ReviewLog, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int) ([]ReviewLog, int, error)
	CountByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (int, error)
	GetReviewCountsPerDay(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]DailyReviewCount, error)
}

type DailyReviewCount struct {
	Date    time.Time `json:"date"`
	Count   int       `json:"count"`
	Correct int       `json:"correct"`
}
