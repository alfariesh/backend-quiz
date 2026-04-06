package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Deck struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	IsArchived     bool      `json:"is_archived"`
	NewCardsPerDay *int      `json:"new_cards_per_day,omitempty"`
	Position       int       `json:"position"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DeckWithCounts struct {
	Deck
	CardCount    int `json:"card_count"`
	NewCount     int `json:"new_count"`
	DueCount     int `json:"due_count"`
	LearnCount   int `json:"learn_count"`
	RelearnCount int `json:"relearn_count"`
}

type DeckShare struct {
	ID        uuid.UUID `json:"id"`
	DeckID    uuid.UUID `json:"deck_id"`
	ShareCode string    `json:"share_code"`
	IsPublic  bool      `json:"is_public"`
	CreatedAt time.Time `json:"created_at"`
}

type DeckRepository interface {
	Create(ctx context.Context, deck *Deck) error
	GetByID(ctx context.Context, id uuid.UUID) (*Deck, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]DeckWithCounts, int, error)
	Update(ctx context.Context, deck *Deck) error
	Delete(ctx context.Context, id uuid.UUID) error

	CreateShare(ctx context.Context, share *DeckShare) error
	GetShareByDeckID(ctx context.Context, deckID uuid.UUID) (*DeckShare, error)
	GetShareByCode(ctx context.Context, code string) (*DeckShare, error)
	DeleteShare(ctx context.Context, deckID uuid.UUID) error
	ListPublicDecks(ctx context.Context, search string, limit, offset int) ([]DeckWithCounts, int, error)

	CloneDeck(ctx context.Context, sourceDeckID, targetUserID uuid.UUID, name string) (*Deck, error)
}
