package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type StartSessionRequest struct {
	DeckID uuid.UUID `json:"deck_id" validate:"required"`
}

type StartSessionResponse struct {
	Session *domain.StudySession `json:"session"`
	Cards   []domain.Card        `json:"cards"`
	Counts  DueCounts            `json:"counts"`
}

type DueCounts struct {
	New      int `json:"new"`
	Learning int `json:"learning"`
	Review   int `json:"review"`
	Total    int `json:"total"`
}

// FSRSCardState represents the pre-computed FSRS card state from the client (ts-fsrs).
type FSRSCardState struct {
	Due           time.Time `json:"due" validate:"required"`
	Stability     float64   `json:"stability" validate:"min=0"`
	Difficulty    float64   `json:"difficulty" validate:"min=0,max=10"`
	ElapsedDays   int       `json:"elapsed_days" validate:"min=0"`
	ScheduledDays int       `json:"scheduled_days" validate:"min=0"`
	Reps          int       `json:"reps" validate:"min=0"`
	Lapses        int       `json:"lapses" validate:"min=0"`
	State         int       `json:"state" validate:"min=0,max=3"`
	LastReview    time.Time `json:"last_review" validate:"required"`
}

// FSRSLogState represents the pre-computed FSRS review log state from the client.
type FSRSLogState struct {
	ScheduledDays int     `json:"scheduled_days"`
	ElapsedDays   int     `json:"elapsed_days"`
	Stability     float64 `json:"stability"`
	Difficulty    float64 `json:"difficulty"`
}

type SubmitReviewRequest struct {
	CardID     uuid.UUID     `json:"card_id" validate:"required"`
	Rating     int           `json:"rating" validate:"required,min=1,max=4"`
	DurationMS int           `json:"duration_ms" validate:"min=0"`
	Card       FSRSCardState `json:"card" validate:"required"`
	Log        FSRSLogState  `json:"log"`
}

type ReviewResult struct {
	Card      domain.Card      `json:"card"`
	ReviewLog domain.ReviewLog `json:"review_log"`
	NextDue   time.Time        `json:"next_due"`
}

type BatchReviewRequest struct {
	Reviews []BatchReviewItem `json:"reviews" validate:"required,min=1,max=500,dive"`
}

type BatchReviewItem struct {
	CardID     uuid.UUID     `json:"card_id" validate:"required"`
	Rating     int           `json:"rating" validate:"required,min=1,max=4"`
	DurationMS int           `json:"duration_ms" validate:"min=0"`
	ReviewedAt time.Time     `json:"reviewed_at" validate:"required"`
	Card       FSRSCardState `json:"card" validate:"required"`
	Log        FSRSLogState  `json:"log"`
}

type BatchReviewResult struct {
	Processed int `json:"processed"`
	Errors    int `json:"errors"`
}

type ReminderResponse struct {
	Decks        []domain.DeckDueSummary `json:"decks"`
	TotalDue     int                     `json:"total_due"`
	DueSoon      int                     `json:"due_soon"`
	Streak       int                     `json:"streak"`
	StudiedToday bool                    `json:"studied_today"`
	NextDueAt    *time.Time              `json:"next_due_at,omitempty"`
}
