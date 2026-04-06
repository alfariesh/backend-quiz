package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CardState int8

const (
	CardStateNew        CardState = 0
	CardStateLearning   CardState = 1
	CardStateReview     CardState = 2
	CardStateRelearning CardState = 3
)

func (s CardState) String() string {
	switch s {
	case CardStateNew:
		return "new"
	case CardStateLearning:
		return "learning"
	case CardStateReview:
		return "review"
	case CardStateRelearning:
		return "relearning"
	default:
		return "unknown"
	}
}

type Rating int8

const (
	RatingAgain Rating = 1
	RatingHard  Rating = 2
	RatingGood  Rating = 3
	RatingEasy  Rating = 4
)

type Card struct {
	ID          uuid.UUID  `json:"id"`
	DeckID      uuid.UUID  `json:"deck_id"`
	Front       string     `json:"front"`
	Back        string     `json:"back"`
	Tags        []string   `json:"tags"`
	IsSuspended bool       `json:"is_suspended"`
	Position    int        `json:"position"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// FSRS state
	Due           time.Time  `json:"due"`
	Stability     float64    `json:"stability"`
	Difficulty    float64    `json:"difficulty"`
	ElapsedDays   int        `json:"elapsed_days"`
	ScheduledDays int        `json:"scheduled_days"`
	Reps          int        `json:"reps"`
	Lapses        int        `json:"lapses"`
	State         CardState  `json:"state"`
	LastReview    *time.Time `json:"last_review,omitempty"`
}

type CardFilter struct {
	State *CardState `json:"state,omitempty"`
	Tag   string     `json:"tag,omitempty"`
	Query string     `json:"query,omitempty"`
}

type CardRepository interface {
	Create(ctx context.Context, card *Card) error
	BulkCreate(ctx context.Context, cards []*Card) error
	GetByID(ctx context.Context, id uuid.UUID) (*Card, error)
	ListByDeckID(ctx context.Context, deckID uuid.UUID, filter CardFilter, limit, offset int) ([]Card, int, error)
	Update(ctx context.Context, card *Card) error
	UpdateFSRS(ctx context.Context, card *Card) error
	Delete(ctx context.Context, id uuid.UUID) error
	SetSuspended(ctx context.Context, id uuid.UUID, suspended bool) error
	ResetFSRS(ctx context.Context, id uuid.UUID) error

	GetDueCards(ctx context.Context, deckID uuid.UUID, now time.Time, newLimit, reviewLimit int) ([]Card, error)
	CountByState(ctx context.Context, deckID uuid.UUID) (map[CardState]int, error)
	CountDue(ctx context.Context, deckID uuid.UUID, now time.Time) (int, error)
}
