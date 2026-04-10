package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

func TestMediaRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	mediaRepo := NewMediaRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "media@example.com")
	deck := createTestDeck(t, pool, user.ID, "Media Deck")
	card := createTestCard(t, pool, deck.ID, "Q", "A")

	media := &domain.Media{
		UserID:   user.ID,
		CardID:   &card.ID,
		FileName: "diagram.png",
		FileSize: 12345,
		MimeType: "image/png",
		R2Key:    "uploads/diagram.png",
		URL:      "https://cdn.example.com/diagram.png",
	}
	err := mediaRepo.Create(ctx, media)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, media.ID)

	got, err := mediaRepo.GetByID(ctx, media.ID)
	require.NoError(t, err)
	assert.Equal(t, "diagram.png", got.FileName)
	assert.Equal(t, 12345, got.FileSize)
	assert.Equal(t, "image/png", got.MimeType)
	assert.Equal(t, "uploads/diagram.png", got.R2Key)
}

func TestMediaRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	mediaRepo := NewMediaRepository(pool)
	ctx := context.Background()

	_, err := mediaRepo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestMediaRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	mediaRepo := NewMediaRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "mediadel@example.com")

	media := &domain.Media{
		UserID:   user.ID,
		FileName: "delete.png",
		FileSize: 100,
		MimeType: "image/png",
		R2Key:    "uploads/delete.png",
		URL:      "https://cdn.example.com/delete.png",
	}
	require.NoError(t, mediaRepo.Create(ctx, media))

	err := mediaRepo.Delete(ctx, media.ID)
	require.NoError(t, err)

	_, err = mediaRepo.GetByID(ctx, media.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestMediaRepo_ListByCardID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	mediaRepo := NewMediaRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "medialist@example.com")
	deck := createTestDeck(t, pool, user.ID, "MediaList Deck")
	card := createTestCard(t, pool, deck.ID, "Q", "A")

	for i, name := range []string{"img1.png", "img2.png"} {
		require.NoError(t, mediaRepo.Create(ctx, &domain.Media{
			UserID:   user.ID,
			CardID:   &card.ID,
			FileName: name,
			FileSize: 100 * (i + 1),
			MimeType: "image/png",
			R2Key:    "uploads/" + name,
			URL:      "https://cdn.example.com/" + name,
		}))
	}

	list, err := mediaRepo.ListByCardID(ctx, card.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}
