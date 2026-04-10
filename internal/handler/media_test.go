package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/alfariesh/backend-quiz/internal/domain"
	mockport "github.com/alfariesh/backend-quiz/internal/mocks/port"
)

func setupMediaRouter(h *MediaHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/cards/{cardID}/media", h.Upload)
	r.Get("/cards/{cardID}/media", h.ListByCard)
	r.Delete("/media/{mediaID}", h.Delete)
	return r
}

// --- Upload ---

func TestMediaHandler_Upload_Success(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	uid := uuid.New()
	cardID := uuid.New()

	svc.EXPECT().Upload(mock.Anything, uid, cardID, mock.Anything).Return(
		&domain.Media{ID: uuid.New(), UserID: uid, FileName: "audio.mp3"}, nil,
	)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "audio.mp3")
	part.Write([]byte("fake-audio"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/cards/"+cardID.String()+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestMediaHandler_Upload_InvalidCardID(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/cards/bad-id/media", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMediaHandler_Upload_NoFile(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	cardID := uuid.New()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/cards/"+cardID.String()+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMediaHandler_Upload_ServiceError(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	uid := uuid.New()
	cardID := uuid.New()

	svc.EXPECT().Upload(mock.Anything, uid, cardID, mock.Anything).Return(nil, domain.ErrForbidden)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "audio.mp3")
	part.Write([]byte("fake-audio"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/cards/"+cardID.String()+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- ListByCard ---

func TestMediaHandler_ListByCard_Success(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	uid := uuid.New()
	cardID := uuid.New()

	svc.EXPECT().ListByCard(mock.Anything, uid, cardID).Return(
		[]domain.Media{{ID: uuid.New(), FileName: "audio.mp3"}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/cards/"+cardID.String()+"/media", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMediaHandler_ListByCard_NilReturnsEmptyArray(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	uid := uuid.New()
	cardID := uuid.New()

	svc.EXPECT().ListByCard(mock.Anything, uid, cardID).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/cards/"+cardID.String()+"/media", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "[]")
}

func TestMediaHandler_ListByCard_InvalidCardID(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/cards/bad-id/media", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMediaHandler_ListByCard_ServiceError(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	uid := uuid.New()
	cardID := uuid.New()
	svc.EXPECT().ListByCard(mock.Anything, uid, cardID).Return(nil, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodGet, "/cards/"+cardID.String()+"/media", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMediaHandler_Upload_InvalidMultipartForm(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	cardID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/cards/"+cardID.String()+"/media", bytes.NewReader([]byte("not multipart")))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Delete ---

func TestMediaHandler_Delete_Success(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	uid := uuid.New()
	mediaID := uuid.New()

	svc.EXPECT().Delete(mock.Anything, uid, mediaID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/media/"+mediaID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "media deleted")
}

func TestMediaHandler_Delete_InvalidID(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/media/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMediaHandler_Delete_NotFound(t *testing.T) {
	svc := mockport.NewMockMediaServicer(t)
	h := NewMediaHandler(svc)
	router := setupMediaRouter(h)

	uid := uuid.New()
	mediaID := uuid.New()

	svc.EXPECT().Delete(mock.Anything, uid, mediaID).Return(domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/media/"+mediaID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
