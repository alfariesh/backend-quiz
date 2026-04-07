package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/repository"
	"github.com/rekanesiads/backend-quiz/internal/service"
)

type MediaHandler struct {
	mediaSvc  *service.MediaService
	mediaRepo *repository.MediaRepository
}

func NewMediaHandler(mediaSvc *service.MediaService, mediaRepo *repository.MediaRepository) *MediaHandler {
	return &MediaHandler{mediaSvc: mediaSvc, mediaRepo: mediaRepo}
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	// Max 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		JSONError(w, http.StatusBadRequest, "file too large or invalid form data")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		JSONError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Upload to R2
	media, err := h.mediaSvc.Upload(r.Context(), userID, header.Filename, mimeType, header.Size, file)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Optionally link to card
	if cardIDStr := r.FormValue("card_id"); cardIDStr != "" {
		if cardID, err := uuid.Parse(cardIDStr); err == nil {
			media.CardID = &cardID
		}
	}

	// Save to database
	if err := h.mediaRepo.Create(r.Context(), media); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, media)
}

func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid media id")
		return
	}

	media, err := h.mediaRepo.GetByID(r.Context(), mediaID)
	if err != nil {
		HandleError(w, err)
		return
	}

	if media.UserID != userID {
		JSONError(w, http.StatusForbidden, "forbidden")
		return
	}

	// Delete from R2
	if err := h.mediaSvc.Delete(r.Context(), media.R2Key); err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to delete file")
		return
	}

	// Delete from database
	if err := h.mediaRepo.Delete(r.Context(), mediaID); err != nil {
		HandleError(w, err)
		return
	}

	JSONMessage(w, http.StatusOK, "media deleted")
}
