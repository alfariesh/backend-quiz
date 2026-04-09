package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	mockdomain "github.com/rekanesiads/backend-quiz/internal/mocks/domain"
)

// --- Create ---

func TestDeckService_Create_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()
	userID := uuid.New()

	deckRepo.On("Create", ctx, mock.AnythingOfType("*domain.Deck")).Return(nil)

	deck, err := svc.Create(ctx, userID, CreateDeckRequest{
		Name:        "My Deck",
		Description: "Test deck",
	})

	require.NoError(t, err)
	assert.Equal(t, "My Deck", deck.Name)
	assert.Equal(t, "Test deck", deck.Description)
	assert.Equal(t, userID, deck.UserID)
}

// --- Get ---

func TestDeckService_Get_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: userID, Name: "Test"}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)

	result, err := svc.Get(ctx, userID, deckID)

	require.NoError(t, err)
	assert.Equal(t, deckID, result.ID)
}

func TestDeckService_Get_Forbidden(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: uuid.New()}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)

	_, err := svc.Get(ctx, uuid.New(), deckID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- List ---

func TestDeckService_List_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()
	decks := []domain.DeckWithCounts{{Deck: domain.Deck{Name: "Deck1"}}}

	deckRepo.On("ListByUserID", ctx, userID, 20, 0).Return(decks, 1, nil)

	result, total, err := svc.List(ctx, userID, 20, 0)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
}

// --- Update ---

func TestDeckService_Update_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: userID, Name: "Old"}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)
	deckRepo.On("Update", ctx, mock.AnythingOfType("*domain.Deck")).Return(nil)

	newName := "New Name"
	newDesc := "New Description"
	archived := true
	result, err := svc.Update(ctx, userID, deckID, UpdateDeckRequest{
		Name:        &newName,
		Description: &newDesc,
		IsArchived:  &archived,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
	assert.Equal(t, "New Description", result.Description)
	assert.True(t, result.IsArchived)
}

func TestDeckService_Update_Forbidden(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: uuid.New()}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)

	name := "hack"
	_, err := svc.Update(ctx, uuid.New(), deckID, UpdateDeckRequest{Name: &name})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- Delete ---

func TestDeckService_Delete_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: userID}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)
	deckRepo.On("Delete", ctx, deckID).Return(nil)

	err := svc.Delete(ctx, userID, deckID)
	assert.NoError(t, err)
}

func TestDeckService_Delete_Forbidden(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: uuid.New()}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)

	err := svc.Delete(ctx, uuid.New(), deckID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- Share ---

func TestDeckService_Share_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: userID}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)
	deckRepo.On("DeleteShare", ctx, deckID).Return(nil)
	deckRepo.On("CreateShare", ctx, mock.AnythingOfType("*domain.DeckShare")).Return(nil)

	share, err := svc.Share(ctx, userID, deckID, ShareDeckRequest{IsPublic: true})

	require.NoError(t, err)
	assert.NotEmpty(t, share.ShareCode)
	assert.True(t, share.IsPublic)
}

func TestDeckService_Share_Forbidden(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: uuid.New()}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)

	_, err := svc.Share(ctx, uuid.New(), deckID, ShareDeckRequest{})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- Unshare ---

func TestDeckService_Unshare_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: userID}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)
	deckRepo.On("DeleteShare", ctx, deckID).Return(nil)

	err := svc.Unshare(ctx, userID, deckID)
	assert.NoError(t, err)
}

// --- Clone ---

func TestDeckService_Clone_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()
	sourceDeckID := uuid.New()
	clonedDeck := &domain.Deck{ID: uuid.New(), UserID: userID, Name: "Source Deck"}

	share := &domain.DeckShare{DeckID: sourceDeckID, ShareCode: "abc123"}
	sourceDeck := &domain.Deck{ID: sourceDeckID, UserID: uuid.New(), Name: "Source Deck"}

	deckRepo.On("GetShareByCode", ctx, "abc123").Return(share, nil)
	deckRepo.On("GetByID", ctx, sourceDeckID).Return(sourceDeck, nil)
	deckRepo.On("CloneDeck", ctx, sourceDeckID, userID, "Source Deck").Return(clonedDeck, nil)

	result, err := svc.Clone(ctx, userID, "abc123")

	require.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
}

func TestDeckService_Clone_InvalidShareCode(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	deckRepo.On("GetShareByCode", ctx, "invalid").Return(nil, domain.ErrNotFound)

	_, err := svc.Clone(ctx, uuid.New(), "invalid")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- ListPublic ---

func TestDeckService_ListPublic_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	decks := []domain.DeckWithCounts{{Deck: domain.Deck{Name: "Public Deck"}}}
	deckRepo.On("ListPublicDecks", ctx, "search", 20, 0).Return(decks, 1, nil)

	result, total, err := svc.ListPublic(ctx, "search", 20, 0)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
}

// --- Export ---

func TestDeckService_Export_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: userID, Name: "Export Deck", Description: "desc"}

	cards := []domain.Card{
		{Front: "Q1", Back: "A1", Tags: []string{"tag1"}},
		{Front: "Q2", Back: "A2", Tags: []string{}},
	}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, 2, nil)

	data, err := svc.Export(ctx, userID, deckID)

	require.NoError(t, err)

	var export ExportDeck
	require.NoError(t, json.Unmarshal(data, &export))
	assert.Equal(t, "Export Deck", export.Name)
	assert.Len(t, export.Cards, 2)
	assert.Equal(t, "Q1", export.Cards[0].Front)
}

func TestDeckService_Export_Forbidden(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	deckID := uuid.New()
	deck := &domain.Deck{ID: deckID, UserID: uuid.New()}

	deckRepo.On("GetByID", ctx, deckID).Return(deck, nil)

	_, err := svc.Export(ctx, uuid.New(), deckID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- Import ---

func TestDeckService_Import_Success(t *testing.T) {
	deckRepo := mockdomain.NewMockDeckRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	svc := NewDeckService(deckRepo, cardRepo)
	ctx := context.Background()

	userID := uuid.New()

	deckRepo.On("Create", ctx, mock.AnythingOfType("*domain.Deck")).Return(nil)
	cardRepo.On("BulkCreate", ctx, mock.AnythingOfType("[]*domain.Card")).Return(nil)

	deck, err := svc.Import(ctx, userID, ImportDeckRequest{
		Name: "Imported Deck",
		Cards: []ExportCard{
			{Front: "Q1", Back: "A1", Tags: []string{"t1"}},
			{Front: "Q2", Back: "A2"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Imported Deck", deck.Name)
	assert.Equal(t, userID, deck.UserID)

	// Verify BulkCreate was called with correct cards
	createCall := cardRepo.Calls[0]
	createdCards := createCall.Arguments.Get(1).([]*domain.Card)
	assert.Len(t, createdCards, 2)
	assert.Equal(t, []string{"t1"}, createdCards[0].Tags)
	assert.Equal(t, []string{}, createdCards[1].Tags) // nil tags default to empty
	assert.Equal(t, 0, createdCards[0].Position)
	assert.Equal(t, 1, createdCards[1].Position)
}
