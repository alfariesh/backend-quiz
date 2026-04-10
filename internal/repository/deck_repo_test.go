package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

// createTestUser is a helper that inserts a user and returns it.
func createTestUser(t *testing.T, repo *UserRepository, email string) *domain.User {
	t.Helper()
	user := &domain.User{
		Email:        email,
		PasswordHash: "hashed",
		DisplayName:  "Test",
		Timezone:     "UTC",
	}
	require.NoError(t, repo.Create(context.Background(), user))
	return user
}

func TestDeckRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	deckRepo := NewDeckRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "deck@example.com")

	deck := &domain.Deck{
		UserID:      user.ID,
		Name:        "Fiqh",
		Description: "Islamic jurisprudence",
	}
	err := deckRepo.Create(ctx, deck)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, deck.ID)

	got, err := deckRepo.GetByID(ctx, deck.ID)
	require.NoError(t, err)
	assert.Equal(t, "Fiqh", got.Name)
	assert.Equal(t, user.ID, got.UserID)
}

func TestDeckRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	deckRepo := NewDeckRepository(pool)
	ctx := context.Background()

	_, err := deckRepo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeckRepo_ListByUserID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	deckRepo := NewDeckRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "decklist@example.com")

	for _, name := range []string{"Deck A", "Deck B", "Deck C"} {
		require.NoError(t, deckRepo.Create(ctx, &domain.Deck{UserID: user.ID, Name: name}))
	}

	decks, total, err := deckRepo.ListByUserID(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, decks, 3)
}

func TestDeckRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	deckRepo := NewDeckRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "deckupd@example.com")

	deck := &domain.Deck{UserID: user.ID, Name: "Old Name", Description: "old"}
	require.NoError(t, deckRepo.Create(ctx, deck))

	deck.Name = "New Name"
	deck.Description = "new"
	err := deckRepo.Update(ctx, deck)
	require.NoError(t, err)

	got, err := deckRepo.GetByID(ctx, deck.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", got.Name)
	assert.Equal(t, "new", got.Description)
}

func TestDeckRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	deckRepo := NewDeckRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "deckdel@example.com")

	deck := &domain.Deck{UserID: user.ID, Name: "Delete Me"}
	require.NoError(t, deckRepo.Create(ctx, deck))

	err := deckRepo.Delete(ctx, deck.ID)
	require.NoError(t, err)

	_, err = deckRepo.GetByID(ctx, deck.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
