package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type CardService struct {
	cardRepo domain.CardRepository
	deckRepo domain.DeckRepository
}

func NewCardService(cardRepo domain.CardRepository, deckRepo domain.DeckRepository) *CardService {
	return &CardService{cardRepo: cardRepo, deckRepo: deckRepo}
}

type CreateCardRequest struct {
	Front string   `json:"front" validate:"required,min=1"`
	Back  string   `json:"back" validate:"required,min=1"`
	Tags  []string `json:"tags"`
}

type UpdateCardRequest struct {
	Front *string  `json:"front,omitempty" validate:"omitempty,min=1"`
	Back  *string  `json:"back,omitempty" validate:"omitempty,min=1"`
	Tags  []string `json:"tags,omitempty"`
}

type BatchCreateRequest struct {
	Cards []CreateCardRequest `json:"cards" validate:"required,min=1,max=500,dive"`
}

type SuspendRequest struct {
	Suspended bool `json:"suspended"`
}

func (s *CardService) Create(ctx context.Context, userID, deckID uuid.UUID, req CreateCardRequest) (*domain.Card, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	card := &domain.Card{
		DeckID: deckID,
		Front:  req.Front,
		Back:   req.Back,
		Tags:   tags,
	}

	if err := s.cardRepo.Create(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *CardService) BatchCreate(ctx context.Context, userID, deckID uuid.UUID, req BatchCreateRequest) ([]*domain.Card, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	cards := make([]*domain.Card, len(req.Cards))
	for i, c := range req.Cards {
		tags := c.Tags
		if tags == nil {
			tags = []string{}
		}
		cards[i] = &domain.Card{
			DeckID:   deckID,
			Front:    c.Front,
			Back:     c.Back,
			Tags:     tags,
			Position: i,
		}
	}

	if err := s.cardRepo.BulkCreate(ctx, cards); err != nil {
		return nil, err
	}
	return cards, nil
}

func (s *CardService) Get(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
	return s.cardRepo.GetByID(ctx, cardID)
}

func (s *CardService) List(ctx context.Context, userID, deckID uuid.UUID, filter domain.CardFilter, limit, offset int) ([]domain.Card, int, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return nil, 0, err
	}
	if deck.UserID != userID {
		return nil, 0, domain.ErrForbidden
	}

	return s.cardRepo.ListByDeckID(ctx, deckID, filter, limit, offset)
}

func (s *CardService) Update(ctx context.Context, userID, cardID uuid.UUID, req UpdateCardRequest) (*domain.Card, error) {
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		return nil, err
	}

	// Verify ownership via deck
	deck, err := s.deckRepo.GetByID(ctx, card.DeckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if req.Front != nil {
		card.Front = *req.Front
	}
	if req.Back != nil {
		card.Back = *req.Back
	}
	if req.Tags != nil {
		card.Tags = req.Tags
	}

	if err := s.cardRepo.Update(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *CardService) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		return err
	}

	deck, err := s.deckRepo.GetByID(ctx, card.DeckID)
	if err != nil {
		return err
	}
	if deck.UserID != userID {
		return domain.ErrForbidden
	}

	return s.cardRepo.Delete(ctx, cardID)
}

func (s *CardService) Suspend(ctx context.Context, userID, cardID uuid.UUID, suspended bool) error {
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		return err
	}

	deck, err := s.deckRepo.GetByID(ctx, card.DeckID)
	if err != nil {
		return err
	}
	if deck.UserID != userID {
		return domain.ErrForbidden
	}

	return s.cardRepo.SetSuspended(ctx, cardID, suspended)
}
