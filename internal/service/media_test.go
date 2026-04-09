package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/config"
	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	mockdomain "github.com/rekanesiads/backend-quiz/internal/mocks/domain"
)

func newTestMediaService(t *testing.T) (
	*MediaService,
	*mockdomain.MockMediaRepository,
	*mockdomain.MockCardRepository,
	*mockdomain.MockDeckRepository,
	*mockdomain.MockObjectStore,
) {
	mediaRepo := mockdomain.NewMockMediaRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	store := mockdomain.NewMockObjectStore(t)
	r2Cfg := config.R2Config{MaxFileSizeMB: 10}
	svc := NewMediaService(mediaRepo, cardRepo, deckRepo, store, r2Cfg)
	return svc, mediaRepo, cardRepo, deckRepo, store
}

// --- Upload ---

func TestMediaService_Upload_Success(t *testing.T) {
	svc, mediaRepo, cardRepo, deckRepo, store := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	store.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, "audio/mpeg").Return("https://cdn.example.com/file.mp3", nil)
	mediaRepo.On("Create", ctx, mock.AnythingOfType("*domain.Media")).Return(nil)

	media, err := svc.Upload(ctx, userID, cardID, dto.UploadMediaRequest{
		FileName:    "recitation.mp3",
		FileSize:    1024,
		ContentType: "audio/mpeg",
		Body:        strings.NewReader("fake-audio-data"),
	})

	require.NoError(t, err)
	assert.Equal(t, userID, media.UserID)
	assert.Equal(t, &cardID, media.CardID)
	assert.Equal(t, "recitation.mp3", media.FileName)
	assert.Equal(t, "audio/mpeg", media.MimeType)
	assert.Contains(t, media.R2Key, "media/")
}

func TestMediaService_Upload_CardNotFound(t *testing.T) {
	svc, _, cardRepo, _, _ := newTestMediaService(t)
	ctx := context.Background()

	cardID := uuid.New()
	cardRepo.On("GetByID", ctx, cardID).Return(nil, domain.ErrNotFound)

	_, err := svc.Upload(ctx, uuid.New(), cardID, dto.UploadMediaRequest{
		FileName: "file.mp3", FileSize: 100, ContentType: "audio/mpeg",
		Body: strings.NewReader("data"),
	})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestMediaService_Upload_Forbidden(t *testing.T) {
	svc, _, cardRepo, deckRepo, _ := newTestMediaService(t)
	ctx := context.Background()

	cardID := uuid.New()
	deckID := uuid.New()
	otherUser := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: otherUser}, nil)

	_, err := svc.Upload(ctx, uuid.New(), cardID, dto.UploadMediaRequest{
		FileName: "file.mp3", FileSize: 100, ContentType: "audio/mpeg",
		Body: strings.NewReader("data"),
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestMediaService_Upload_FileTooLarge(t *testing.T) {
	svc, _, cardRepo, deckRepo, _ := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	_, err := svc.Upload(ctx, userID, cardID, dto.UploadMediaRequest{
		FileName: "huge.mp3", FileSize: 11 * 1024 * 1024, ContentType: "audio/mpeg",
		Body: strings.NewReader("data"),
	})
	assert.ErrorIs(t, err, domain.ErrFileTooLarge)
}

func TestMediaService_Upload_UnsupportedMedia(t *testing.T) {
	svc, _, cardRepo, deckRepo, _ := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	_, err := svc.Upload(ctx, userID, cardID, dto.UploadMediaRequest{
		FileName: "doc.pdf", FileSize: 100, ContentType: "application/pdf",
		Body: strings.NewReader("data"),
	})
	assert.ErrorIs(t, err, domain.ErrUnsupportedMedia)
}

func TestMediaService_Upload_NoExtension_Audio(t *testing.T) {
	svc, mediaRepo, cardRepo, deckRepo, store := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	store.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, "audio/mpeg").Return("https://cdn.example.com/file.mp3", nil)
	mediaRepo.On("Create", ctx, mock.AnythingOfType("*domain.Media")).Return(nil)

	media, err := svc.Upload(ctx, userID, cardID, dto.UploadMediaRequest{
		FileName: "noext", FileSize: 100, ContentType: "audio/mpeg",
		Body: strings.NewReader("data"),
	})
	require.NoError(t, err)
	assert.Contains(t, media.R2Key, ".mp3")
}

func TestMediaService_Upload_NoExtension_NonAudio(t *testing.T) {
	svc, mediaRepo, cardRepo, deckRepo, store := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	store.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, "image/png").Return("https://cdn.example.com/file.bin", nil)
	mediaRepo.On("Create", ctx, mock.AnythingOfType("*domain.Media")).Return(nil)

	media, err := svc.Upload(ctx, userID, cardID, dto.UploadMediaRequest{
		FileName: "noext", FileSize: 100, ContentType: "image/png",
		Body: strings.NewReader("data"),
	})
	require.NoError(t, err)
	assert.Contains(t, media.R2Key, ".bin")
}

func TestMediaService_Upload_StoreError(t *testing.T) {
	svc, _, cardRepo, deckRepo, store := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	store.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, "audio/mpeg").Return("", errors.New("s3 down"))

	_, err := svc.Upload(ctx, userID, cardID, dto.UploadMediaRequest{
		FileName: "file.mp3", FileSize: 100, ContentType: "audio/mpeg",
		Body: strings.NewReader("data"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uploading file")
}

func TestMediaService_Upload_DBError_CleansUpR2(t *testing.T) {
	svc, mediaRepo, cardRepo, deckRepo, store := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	store.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, "audio/mpeg").Return("https://cdn.example.com/file.mp3", nil)
	mediaRepo.On("Create", ctx, mock.AnythingOfType("*domain.Media")).Return(errors.New("db error"))
	store.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)

	_, err := svc.Upload(ctx, userID, cardID, dto.UploadMediaRequest{
		FileName: "file.mp3", FileSize: 100, ContentType: "audio/mpeg",
		Body: strings.NewReader("data"),
	})
	assert.Error(t, err)
	store.AssertCalled(t, "Delete", ctx, mock.AnythingOfType("string"))
}

// --- Delete ---

func TestMediaService_Delete_Success(t *testing.T) {
	svc, mediaRepo, _, _, store := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	mediaID := uuid.New()

	mediaRepo.On("GetByID", ctx, mediaID).Return(&domain.Media{
		ID: mediaID, UserID: userID, R2Key: "media/key.mp3",
	}, nil)
	store.On("Delete", ctx, "media/key.mp3").Return(nil)
	mediaRepo.On("Delete", ctx, mediaID).Return(nil)

	err := svc.Delete(ctx, userID, mediaID)
	assert.NoError(t, err)
}

func TestMediaService_Delete_NotFound(t *testing.T) {
	svc, mediaRepo, _, _, _ := newTestMediaService(t)
	ctx := context.Background()

	mediaID := uuid.New()
	mediaRepo.On("GetByID", ctx, mediaID).Return(nil, domain.ErrNotFound)

	err := svc.Delete(ctx, uuid.New(), mediaID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestMediaService_Delete_Forbidden(t *testing.T) {
	svc, mediaRepo, _, _, _ := newTestMediaService(t)
	ctx := context.Background()

	mediaID := uuid.New()
	mediaRepo.On("GetByID", ctx, mediaID).Return(&domain.Media{
		ID: mediaID, UserID: uuid.New(), R2Key: "media/key.mp3",
	}, nil)

	err := svc.Delete(ctx, uuid.New(), mediaID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestMediaService_Delete_StoreError(t *testing.T) {
	svc, mediaRepo, _, _, store := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	mediaID := uuid.New()

	mediaRepo.On("GetByID", ctx, mediaID).Return(&domain.Media{
		ID: mediaID, UserID: userID, R2Key: "media/key.mp3",
	}, nil)
	store.On("Delete", ctx, "media/key.mp3").Return(errors.New("s3 down"))

	err := svc.Delete(ctx, userID, mediaID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deleting file")
}

// --- ListByCard ---

func TestMediaService_ListByCard_Success(t *testing.T) {
	svc, mediaRepo, cardRepo, deckRepo, _ := newTestMediaService(t)
	ctx := context.Background()

	userID := uuid.New()
	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	mediaRepo.On("ListByCardID", ctx, cardID).Return([]domain.Media{
		{ID: uuid.New(), FileName: "audio.mp3"},
	}, nil)

	list, err := svc.ListByCard(ctx, userID, cardID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestMediaService_ListByCard_Forbidden(t *testing.T) {
	svc, _, cardRepo, deckRepo, _ := newTestMediaService(t)
	ctx := context.Background()

	cardID := uuid.New()
	deckID := uuid.New()

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, DeckID: deckID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: uuid.New()}, nil)

	_, err := svc.ListByCard(ctx, uuid.New(), cardID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestMediaService_ListByCard_CardNotFound(t *testing.T) {
	svc, _, cardRepo, _, _ := newTestMediaService(t)
	ctx := context.Background()

	cardID := uuid.New()
	cardRepo.On("GetByID", ctx, cardID).Return(nil, domain.ErrNotFound)

	_, err := svc.ListByCard(ctx, uuid.New(), cardID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
