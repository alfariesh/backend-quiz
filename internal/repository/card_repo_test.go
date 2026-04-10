package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

func TestCardRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "card@example.com")
	deck := createTestDeck(t, pool, user.ID, "Test Deck")

	card := &domain.Card{
		DeckID:      deck.ID,
		Front:       "What is Tawhid?",
		Back:        "Oneness of God",
		ContentType: domain.ContentTypePlain,
		Tags:        []string{"aqidah", "basics"},
	}
	err := cardRepo.Create(ctx, card)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, card.ID)

	got, err := cardRepo.GetByID(ctx, card.ID)
	require.NoError(t, err)
	assert.Equal(t, "What is Tawhid?", got.Front)
	assert.Equal(t, "Oneness of God", got.Back)
	assert.Equal(t, domain.ContentTypePlain, got.ContentType)
	assert.Equal(t, []string{"aqidah", "basics"}, got.Tags)
	assert.Equal(t, domain.CardStateNew, got.State)
}

func TestCardRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	_, err := cardRepo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCardRepo_BulkCreate(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "cardbulk@example.com")
	deck := createTestDeck(t, pool, user.ID, "Bulk Deck")

	cards := []*domain.Card{
		{DeckID: deck.ID, Front: "Q1", Back: "A1", Tags: []string{}},
		{DeckID: deck.ID, Front: "Q2", Back: "A2", Tags: []string{}},
		{DeckID: deck.ID, Front: "Q3", Back: "A3", Tags: []string{}},
	}
	err := cardRepo.BulkCreate(ctx, cards)
	require.NoError(t, err)

	for _, c := range cards {
		assert.NotEqual(t, uuid.Nil, c.ID)
	}

	listed, total, err := cardRepo.ListByDeckID(ctx, deck.ID, domain.CardFilter{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, listed, 3)
}

func TestCardRepo_ListByDeckID_WithFilter(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "cardfilter@example.com")
	deck := createTestDeck(t, pool, user.ID, "Filter Deck")

	require.NoError(t, cardRepo.Create(ctx, &domain.Card{DeckID: deck.ID, Front: "Tagged Q", Back: "A", Tags: []string{"fiqh"}}))
	require.NoError(t, cardRepo.Create(ctx, &domain.Card{DeckID: deck.ID, Front: "Other Q", Back: "B", Tags: []string{"aqidah"}}))

	// Filter by tag
	cards, total, err := cardRepo.ListByDeckID(ctx, deck.ID, domain.CardFilter{Tag: "fiqh"}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, cards, 1)
	assert.Equal(t, "Tagged Q", cards[0].Front)
}

func TestCardRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "cardupd@example.com")
	deck := createTestDeck(t, pool, user.ID, "Update Deck")

	card := createTestCard(t, pool, deck.ID, "Old Front", "Old Back")

	card.Front = "New Front"
	card.Back = "New Back"
	card.ContentType = domain.ContentTypeMarkdown
	err := cardRepo.Update(ctx, card)
	require.NoError(t, err)

	got, err := cardRepo.GetByID(ctx, card.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Front", got.Front)
	assert.Equal(t, "New Back", got.Back)
	assert.Equal(t, domain.ContentTypeMarkdown, got.ContentType)
}

func TestCardRepo_UpdateFSRS(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "cardfsrs@example.com")
	deck := createTestDeck(t, pool, user.ID, "FSRS Deck")

	card := createTestCard(t, pool, deck.ID, "FSRS Q", "FSRS A")

	now := time.Now().UTC().Truncate(time.Microsecond)
	card.Due = now.Add(24 * time.Hour)
	card.Stability = 5.5
	card.Difficulty = 3.2
	card.ElapsedDays = 1
	card.ScheduledDays = 7
	card.Reps = 1
	card.Lapses = 0
	card.State = domain.CardStateReview
	card.LastReview = &now

	err := cardRepo.UpdateFSRS(ctx, card)
	require.NoError(t, err)

	got, err := cardRepo.GetByID(ctx, card.ID)
	require.NoError(t, err)
	assert.InDelta(t, 5.5, got.Stability, 0.01)
	assert.InDelta(t, 3.2, got.Difficulty, 0.01)
	assert.Equal(t, 1, got.Reps)
	assert.Equal(t, domain.CardStateReview, got.State)
	assert.NotNil(t, got.LastReview)
}

func TestCardRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "carddel@example.com")
	deck := createTestDeck(t, pool, user.ID, "Delete Deck")

	card := createTestCard(t, pool, deck.ID, "Delete Me", "Gone")

	err := cardRepo.Delete(ctx, card.ID)
	require.NoError(t, err)

	_, err = cardRepo.GetByID(ctx, card.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCardRepo_SetSuspended(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "cardsusp@example.com")
	deck := createTestDeck(t, pool, user.ID, "Suspend Deck")

	card := createTestCard(t, pool, deck.ID, "Suspend Me", "Answer")
	assert.False(t, card.IsSuspended)

	err := cardRepo.SetSuspended(ctx, card.ID, true)
	require.NoError(t, err)

	got, err := cardRepo.GetByID(ctx, card.ID)
	require.NoError(t, err)
	assert.True(t, got.IsSuspended)
}

func TestCardRepo_ResetFSRS(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "cardreset@example.com")
	deck := createTestDeck(t, pool, user.ID, "Reset Deck")

	card := createTestCard(t, pool, deck.ID, "Reset Me", "Answer")

	// Set some FSRS state first
	now := time.Now().UTC()
	card.Stability = 10.0
	card.Reps = 5
	card.State = domain.CardStateReview
	card.LastReview = &now
	require.NoError(t, cardRepo.UpdateFSRS(ctx, card))

	// Reset
	err := cardRepo.ResetFSRS(ctx, card.ID)
	require.NoError(t, err)

	got, err := cardRepo.GetByID(ctx, card.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.CardStateNew, got.State)
	assert.Equal(t, 0, got.Reps)
	assert.InDelta(t, 0, got.Stability, 0.01)
}

func TestCardRepo_CountByState(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "cardcount@example.com")
	deck := createTestDeck(t, pool, user.ID, "Count Deck")

	// Create 3 new cards
	for i := 0; i < 3; i++ {
		createTestCard(t, pool, deck.ID, "Q", "A")
	}

	counts, err := cardRepo.CountByState(ctx, deck.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, counts[domain.CardStateNew])
}

func TestCardRepo_GetDueCards(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "carddue@example.com")
	deck := createTestDeck(t, pool, user.ID, "Due Deck")

	// New cards have due = zero time, so they should be "due"
	for i := 0; i < 5; i++ {
		createTestCard(t, pool, deck.ID, "Q", "A")
	}

	cards, err := cardRepo.GetDueCards(ctx, deck.ID, time.Now().UTC().Add(time.Hour), 3, 10)
	require.NoError(t, err)
	// New card limit = 3, so at most 3 new cards
	assert.LessOrEqual(t, len(cards), 3)
}

func TestCardRepo_CountDue(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	cardRepo := NewCardRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "cardcountdue@example.com")
	deck := createTestDeck(t, pool, user.ID, "CountDue Deck")

	// CountDue only counts Learning/Review/Relearning cards (not New)
	// So create cards and set them to Review state
	for i := 0; i < 4; i++ {
		card := createTestCard(t, pool, deck.ID, "Q", "A")
		card.State = domain.CardStateReview
		card.Stability = 5.0
		card.Difficulty = 5.0
		card.Reps = 1
		card.Due = time.Now().UTC().Add(-time.Hour) // already due
		require.NoError(t, cardRepo.UpdateFSRS(ctx, card))
	}

	count, err := cardRepo.CountDue(ctx, deck.ID, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}
