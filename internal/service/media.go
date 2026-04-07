package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/config"
	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/pkg/storage"
)

var allowedAudioTypes = map[string]bool{
	"audio/mpeg": true,
	"audio/wav":  true,
	"audio/ogg":  true,
	"audio/mp4":  true,
	"audio/webm": true,
}

type MediaService struct {
	mediaRepo domain.MediaRepository
	cardRepo  domain.CardRepository
	deckRepo  domain.DeckRepository
	r2        *storage.R2Client
	r2Config  config.R2Config
}

func NewMediaService(
	mediaRepo domain.MediaRepository,
	cardRepo domain.CardRepository,
	deckRepo domain.DeckRepository,
	r2 *storage.R2Client,
	r2Config config.R2Config,
) *MediaService {
	return &MediaService{
		mediaRepo: mediaRepo,
		cardRepo:  cardRepo,
		deckRepo:  deckRepo,
		r2:        r2,
		r2Config:  r2Config,
	}
}

type UploadMediaRequest struct {
	FileName    string
	FileSize    int
	ContentType string
	Body        io.Reader
}

func (s *MediaService) Upload(ctx context.Context, userID, cardID uuid.UUID, req UploadMediaRequest) (*domain.Media, error) {
	// Validate card ownership
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		return nil, err
	}
	deck, err := s.deckRepo.GetByID(ctx, card.DeckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	// Validate file size
	maxBytes := s.r2Config.MaxFileSizeMB * 1024 * 1024
	if req.FileSize > maxBytes {
		return nil, domain.ErrFileTooLarge
	}

	// Validate content type
	if !allowedAudioTypes[strings.ToLower(req.ContentType)] {
		return nil, domain.ErrUnsupportedMedia
	}

	// Generate R2 key
	ext := filepath.Ext(req.FileName)
	if ext == "" {
		ext = ".mp3"
	}
	fileID := uuid.New()
	r2Key := fmt.Sprintf("media/%s/%s%s", userID.String(), fileID.String(), ext)

	// Upload to R2
	url, err := s.r2.Upload(ctx, r2Key, req.Body, req.ContentType)
	if err != nil {
		return nil, fmt.Errorf("uploading file: %w", err)
	}

	// Save to database
	media := &domain.Media{
		UserID:   userID,
		CardID:   &cardID,
		FileName: req.FileName,
		FileSize: req.FileSize,
		MimeType: req.ContentType,
		R2Key:    r2Key,
		URL:      url,
	}
	if err := s.mediaRepo.Create(ctx, media); err != nil {
		// Clean up R2 on DB failure
		_ = s.r2.Delete(ctx, r2Key)
		return nil, err
	}

	return media, nil
}

func (s *MediaService) Delete(ctx context.Context, userID, mediaID uuid.UUID) error {
	media, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}
	if media.UserID != userID {
		return domain.ErrForbidden
	}

	// Delete from R2
	if err := s.r2.Delete(ctx, media.R2Key); err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}

	return s.mediaRepo.Delete(ctx, mediaID)
}

func (s *MediaService) ListByCard(ctx context.Context, userID, cardID uuid.UUID) ([]domain.Media, error) {
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		return nil, err
	}
	deck, err := s.deckRepo.GetByID(ctx, card.DeckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return s.mediaRepo.ListByCardID(ctx, cardID)
}
