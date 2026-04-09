package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	mockdomain "github.com/rekanesiads/backend-quiz/internal/mocks/domain"
)

var (
	testUserID = uuid.New()
	testDeckID = uuid.New()
	testCardID = uuid.New()
)

func testDeck(userID, deckID uuid.UUID) *domain.Deck {
	return &domain.Deck{ID: deckID, UserID: userID, Name: "Test Deck"}
}

func testCard(cardID, deckID uuid.UUID) *domain.Card {
	return &domain.Card{
		ID:          cardID,
		DeckID:      deckID,
		Front:       "front",
		Back:        "back",
		ContentType: domain.ContentTypePlain,
		Tags:        []string{},
	}
}

// --- Create ---

func TestCardService_Create_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("Create", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)

	card, err := svc.Create(ctx, testUserID, testDeckID, dto.CreateCardRequest{
		Front: "What is Go?",
		Back:  "A programming language",
		Tags:  []string{"programming"},
	})

	require.NoError(t, err)
	assert.Equal(t, "What is Go?", card.Front)
	assert.Equal(t, "A programming language", card.Back)
	assert.Equal(t, domain.ContentTypePlain, card.ContentType)
	assert.Equal(t, []string{"programming"}, card.Tags)
}

func TestCardService_Create_WithContentType(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("Create", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)

	card, err := svc.Create(ctx, testUserID, testDeckID, dto.CreateCardRequest{
		Front:       "# Title",
		Back:        "**bold**",
		ContentType: "markdown",
	})

	require.NoError(t, err)
	assert.Equal(t, "markdown", card.ContentType)
}

func TestCardService_Create_NilTagsDefaultsToEmpty(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("Create", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)

	card, err := svc.Create(ctx, testUserID, testDeckID, dto.CreateCardRequest{
		Front: "Q", Back: "A",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{}, card.Tags)
}

func TestCardService_Create_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	otherUser := uuid.New()
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(otherUser, testDeckID), nil)

	_, err := svc.Create(ctx, testUserID, testDeckID, dto.CreateCardRequest{Front: "Q", Back: "A"})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestCardService_Create_DeckNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	deckRepo.On("GetByID", ctx, testDeckID).Return(nil, domain.ErrNotFound)

	_, err := svc.Create(ctx, testUserID, testDeckID, dto.CreateCardRequest{Front: "Q", Back: "A"})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- BatchCreate ---

func TestCardService_BatchCreate_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("BulkCreate", ctx, mock.AnythingOfType("[]*domain.Card")).Return(nil)

	cards, err := svc.BatchCreate(ctx, testUserID, testDeckID, dto.BatchCreateRequest{
		Cards: []dto.CreateCardRequest{
			{Front: "Q1", Back: "A1"},
			{Front: "Q2", Back: "A2", ContentType: "markdown"},
		},
	})

	require.NoError(t, err)
	require.Len(t, cards, 2)
	assert.Equal(t, domain.ContentTypePlain, cards[0].ContentType)
	assert.Equal(t, "markdown", cards[1].ContentType)
	assert.Equal(t, 0, cards[0].Position)
	assert.Equal(t, 1, cards[1].Position)
}

func TestCardService_BatchCreate_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	otherUser := uuid.New()
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(otherUser, testDeckID), nil)

	_, err := svc.BatchCreate(ctx, testUserID, testDeckID, dto.BatchCreateRequest{
		Cards: []dto.CreateCardRequest{{Front: "Q", Back: "A"}},
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- Get ---

func TestCardService_Get_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)

	result, err := svc.Get(ctx, testCardID)

	require.NoError(t, err)
	assert.Equal(t, testCardID, result.ID)
}

func TestCardService_Get_NotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(nil, domain.ErrNotFound)

	_, err := svc.Get(ctx, testCardID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- List ---

func TestCardService_List_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	filter := domain.CardFilter{}
	cards := []domain.Card{{ID: uuid.New()}, {ID: uuid.New()}}
	cardRepo.On("ListByDeckID", ctx, testDeckID, filter, 20, 0).Return(cards, 2, nil)

	result, total, err := svc.List(ctx, testUserID, testDeckID, filter, 20, 0)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

func TestCardService_List_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	otherUser := uuid.New()
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(otherUser, testDeckID), nil)

	_, _, err := svc.List(ctx, testUserID, testDeckID, domain.CardFilter{}, 20, 0)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- Update ---

func TestCardService_Update_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("Update", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)

	newFront := "Updated front"
	newBack := "Updated back"
	result, err := svc.Update(ctx, testUserID, testCardID, dto.UpdateCardRequest{
		Front: &newFront,
		Back:  &newBack,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated front", result.Front)
	assert.Equal(t, "Updated back", result.Back)
}

func TestCardService_Update_PartialUpdate(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	card := &domain.Card{
		ID: testCardID, DeckID: testDeckID,
		Front: "original", Back: "original back", ContentType: "plain",
	}
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("Update", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)

	newFront := "new front"
	result, err := svc.Update(ctx, testUserID, testCardID, dto.UpdateCardRequest{Front: &newFront})

	require.NoError(t, err)
	assert.Equal(t, "new front", result.Front)
	assert.Equal(t, "original back", result.Back) // unchanged
}

func TestCardService_Update_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	otherUser := uuid.New()
	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(otherUser, testDeckID), nil)

	newFront := "hack"
	_, err := svc.Update(ctx, testUserID, testCardID, dto.UpdateCardRequest{Front: &newFront})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- Delete ---

func TestCardService_Delete_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("Delete", ctx, testCardID).Return(nil)

	err := svc.Delete(ctx, testUserID, testCardID)
	assert.NoError(t, err)
}

func TestCardService_Delete_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	otherUser := uuid.New()
	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(otherUser, testDeckID), nil)

	err := svc.Delete(ctx, testUserID, testCardID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- ResetFSRS ---

func TestCardService_ResetFSRS_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	card := testCard(testCardID, testDeckID)
	card.State = domain.CardStateReview
	card.Stability = 10.0

	resetCard := testCard(testCardID, testDeckID)
	resetCard.State = domain.CardStateNew
	resetCard.Stability = 0

	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil).Once()
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("ResetFSRS", ctx, testCardID).Return(nil)
	cardRepo.On("GetByID", ctx, testCardID).Return(resetCard, nil).Once()

	result, err := svc.ResetFSRS(ctx, testUserID, testCardID)

	require.NoError(t, err)
	assert.Equal(t, domain.CardStateNew, result.State)
	assert.Equal(t, 0.0, result.Stability)
}

func TestCardService_ResetFSRS_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	otherUser := uuid.New()
	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(otherUser, testDeckID), nil)

	_, err := svc.ResetFSRS(ctx, testUserID, testCardID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestCardService_ResetFSRS_CardNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(nil, domain.ErrNotFound)

	_, err := svc.ResetFSRS(ctx, testUserID, testCardID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- Suspend ---

func TestCardService_Suspend_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("SetSuspended", ctx, testCardID, true).Return(nil)

	err := svc.Suspend(ctx, testUserID, testCardID, true)
	assert.NoError(t, err)
}

func TestCardService_Update_CardNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(nil, domain.ErrNotFound)

	front := "Q"
	_, err := svc.Update(ctx, testUserID, testCardID, dto.UpdateCardRequest{Front: &front})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCardService_Update_WithTagsAndContentType(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(testUserID, testDeckID), nil)
	cardRepo.On("Update", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)

	ct := "markdown"
	tags := []string{"quran", "surah"}
	result, err := svc.Update(ctx, testUserID, testCardID, dto.UpdateCardRequest{
		ContentType: &ct,
		Tags:        tags,
	})

	require.NoError(t, err)
	assert.Equal(t, "markdown", result.ContentType)
	assert.Equal(t, []string{"quran", "surah"}, result.Tags)
}

func TestCardService_Delete_CardNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(nil, domain.ErrNotFound)

	err := svc.Delete(ctx, testUserID, testCardID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCardService_Suspend_CardNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(nil, domain.ErrNotFound)

	err := svc.Suspend(ctx, testUserID, testCardID, true)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCardService_List_DeckNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	deckRepo.On("GetByID", ctx, testDeckID).Return(nil, domain.ErrNotFound)

	_, _, err := svc.List(ctx, testUserID, testDeckID, domain.CardFilter{}, 20, 0)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCardService_BatchCreate_DeckNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	deckRepo.On("GetByID", ctx, testDeckID).Return(nil, domain.ErrNotFound)

	_, err := svc.BatchCreate(ctx, testUserID, testDeckID, dto.BatchCreateRequest{
		Cards: []dto.CreateCardRequest{{Front: "Q", Back: "A"}},
	})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCardService_Suspend_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	otherUser := uuid.New()
	card := testCard(testCardID, testDeckID)
	cardRepo.On("GetByID", ctx, testCardID).Return(card, nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(otherUser, testDeckID), nil)

	err := svc.Suspend(ctx, testUserID, testCardID, true)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestCardService_ResetFSRS_DeckForbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(testCard(testCardID, testDeckID), nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(testDeck(uuid.New(), testDeckID), nil)

	_, err := svc.ResetFSRS(ctx, testUserID, testCardID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestCardService_ResetFSRS_DeckError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(testCard(testCardID, testDeckID), nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(nil, assert.AnError)

	_, err := svc.ResetFSRS(ctx, testUserID, testCardID)
	assert.Error(t, err)
}

func TestCardService_Delete_DeckError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(testCard(testCardID, testDeckID), nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(nil, assert.AnError)

	err := svc.Delete(ctx, testUserID, testCardID)
	assert.Error(t, err)
}

func TestCardService_Suspend_DeckError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	svc := NewCardService(cardRepo, deckRepo)
	ctx := context.Background()

	cardRepo.On("GetByID", ctx, testCardID).Return(testCard(testCardID, testDeckID), nil)
	deckRepo.On("GetByID", ctx, testDeckID).Return(nil, assert.AnError)

	err := svc.Suspend(ctx, testUserID, testCardID, true)
	assert.Error(t, err)
}
