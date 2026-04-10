package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/dto"
	"github.com/alfariesh/backend-quiz/internal/port"
)

var _ port.DeckServicer = (*DeckService)(nil)

type DeckService struct {
	deckRepo domain.DeckRepository
	cardRepo domain.CardRepository
	uow      domain.UnitOfWork
}

func NewDeckService(deckRepo domain.DeckRepository, cardRepo domain.CardRepository, uow domain.UnitOfWork) *DeckService {
	return &DeckService{deckRepo: deckRepo, cardRepo: cardRepo, uow: uow}
}

func (s *DeckService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateDeckRequest) (*domain.Deck, error) {
	deck := &domain.Deck{
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		NewCardsPerDay: req.NewCardsPerDay,
	}

	if err := s.deckRepo.Create(ctx, deck); err != nil {
		return nil, err
	}
	return deck, nil
}

func (s *DeckService) Get(ctx context.Context, userID, deckID uuid.UUID) (*domain.Deck, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return deck, nil
}

func (s *DeckService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.DeckWithCounts, int, error) {
	return s.deckRepo.ListByUserID(ctx, userID, limit, offset)
}

func (s *DeckService) Update(ctx context.Context, userID, deckID uuid.UUID, req dto.UpdateDeckRequest) (*domain.Deck, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if req.Name != nil {
		deck.Name = *req.Name
	}
	if req.Description != nil {
		deck.Description = *req.Description
	}
	if req.IsArchived != nil {
		deck.IsArchived = *req.IsArchived
	}
	if req.NewCardsPerDay != nil {
		deck.NewCardsPerDay = req.NewCardsPerDay
	}
	if req.Position != nil {
		deck.Position = *req.Position
	}

	if err := s.deckRepo.Update(ctx, deck); err != nil {
		return nil, err
	}
	return deck, nil
}

func (s *DeckService) Delete(ctx context.Context, userID, deckID uuid.UUID) error {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return err
	}
	if deck.UserID != userID {
		return domain.ErrForbidden
	}
	return s.deckRepo.Delete(ctx, deckID)
}

func (s *DeckService) Share(ctx context.Context, userID, deckID uuid.UUID, req dto.ShareDeckRequest) (*domain.DeckShare, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	// Delete existing share if any
	_ = s.deckRepo.DeleteShare(ctx, deckID)

	code := generateShareCode()
	share := &domain.DeckShare{
		DeckID:    deckID,
		ShareCode: code,
		IsPublic:  req.IsPublic,
	}
	if err := s.deckRepo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *DeckService) Unshare(ctx context.Context, userID, deckID uuid.UUID) error {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return err
	}
	if deck.UserID != userID {
		return domain.ErrForbidden
	}
	return s.deckRepo.DeleteShare(ctx, deckID)
}

func (s *DeckService) Clone(ctx context.Context, userID uuid.UUID, shareCode string) (*domain.Deck, error) {
	share, err := s.deckRepo.GetShareByCode(ctx, shareCode)
	if err != nil {
		return nil, err
	}

	sourceDeck, err := s.deckRepo.GetByID(ctx, share.DeckID)
	if err != nil {
		return nil, err
	}

	return s.deckRepo.CloneDeck(ctx, sourceDeck.ID, userID, sourceDeck.Name)
}

func (s *DeckService) ListPublic(ctx context.Context, search string, limit, offset int) ([]domain.DeckWithCounts, int, error) {
	return s.deckRepo.ListPublicDecks(ctx, search, limit, offset)
}

func (s *DeckService) Export(ctx context.Context, userID, deckID uuid.UUID) ([]byte, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	cards, _, err := s.cardRepo.ListByDeckID(ctx, deckID, domain.CardFilter{}, 10000, 0)
	if err != nil {
		return nil, err
	}

	export := dto.ExportDeck{
		Name:        deck.Name,
		Description: deck.Description,
		Cards:       make([]dto.ExportCard, len(cards)),
	}
	for i, c := range cards {
		export.Cards[i] = dto.ExportCard{Front: c.Front, Back: c.Back, Tags: c.Tags}
	}

	return json.MarshalIndent(export, "", "  ")
}

func (s *DeckService) Import(ctx context.Context, userID uuid.UUID, req dto.ImportDeckRequest) (*domain.Deck, error) {
	deck := &domain.Deck{
		UserID: userID,
		Name:   req.Name,
	}

	cards := make([]*domain.Card, len(req.Cards))
	for i, c := range req.Cards {
		tags := c.Tags
		if tags == nil {
			tags = []string{}
		}
		cards[i] = &domain.Card{
			Front:    c.Front,
			Back:     c.Back,
			Tags:     tags,
			Position: i,
		}
	}

	err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.deckRepo.Create(ctx, deck); err != nil {
			return err
		}
		for _, c := range cards {
			c.DeckID = deck.ID
		}
		return s.cardRepo.BulkCreate(ctx, cards)
	})
	if err != nil {
		return nil, fmt.Errorf("importing deck: %w", err)
	}

	return deck, nil
}

func generateShareCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}
