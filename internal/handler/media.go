package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/port"
)

type MediaHandler struct {
	mediaSvc port.MediaServicer
}

func NewMediaHandler(mediaSvc port.MediaServicer) *MediaHandler {
	return &MediaHandler{mediaSvc: mediaSvc}
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "cardID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	// Parse multipart form (max 10MB + overhead)
	if err := r.ParseMultipartForm(11 << 20); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		JSONError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	req := dto.UploadMediaRequest{
		FileName:    header.Filename,
		FileSize:    int(header.Size),
		ContentType: header.Header.Get("Content-Type"),
		Body:        file,
	}

	media, err := h.mediaSvc.Upload(r.Context(), userID, cardID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, media)
}

func (h *MediaHandler) ListByCard(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "cardID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	media, err := h.mediaSvc.ListByCard(r.Context(), userID, cardID)
	if err != nil {
		HandleError(w, err)
		return
	}
	if media == nil {
		media = []domain.Media{}
	}
	JSON(w, http.StatusOK, media)
}

func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid media id")
		return
	}

	if err := h.mediaSvc.Delete(r.Context(), userID, mediaID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "media deleted")
}
