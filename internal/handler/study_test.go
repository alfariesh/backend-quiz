package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	mockport "github.com/rekanesiads/backend-quiz/internal/mocks/port"
)

func setupStudyRouter(h *StudyHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/study/sessions", h.StartSession)
	r.Get("/study/sessions/{sessionID}", h.GetSession)
	r.Post("/study/sessions/{sessionID}/reviews", h.SubmitReview)
	r.Post("/study/sessions/{sessionID}/reviews/batch", h.BatchReview)
	r.Post("/study/sessions/{sessionID}/end", h.EndSession)
	r.Get("/study/reminders", h.Reminders)
	return r
}

// --- StartSession ---

func TestStudyHandler_StartSession_Success(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().StartSession(mock.Anything, uid, mock.Anything).Return(
		&dto.StartSessionResponse{
			Session: &domain.StudySession{ID: uuid.New(), UserID: uid, DeckID: &deckID},
			Cards:   []domain.Card{{ID: uuid.New(), Front: "Q"}},
			Counts:  dto.DueCounts{New: 1, Total: 1},
		}, nil,
	)

	body := `{"deck_id":"` + deckID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/study/sessions", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Contains(t, data, "session")
	assert.Contains(t, data, "cards")
}

func TestStudyHandler_StartSession_InvalidBody(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/study/sessions", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStudyHandler_StartSession_ValidationError(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	body := `{}` // deck_id required
	req := httptest.NewRequest(http.MethodPost, "/study/sessions", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetSession ---

func TestStudyHandler_GetSession_Success(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	uid := uuid.New()
	sessionID := uuid.New()
	svc.EXPECT().GetSession(mock.Anything, uid, sessionID).Return(
		&domain.StudySession{ID: sessionID, UserID: uid}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/study/sessions/"+sessionID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStudyHandler_GetSession_InvalidID(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/study/sessions/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStudyHandler_GetSession_NotFound(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	uid := uuid.New()
	sessionID := uuid.New()
	svc.EXPECT().GetSession(mock.Anything, uid, sessionID).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/study/sessions/"+sessionID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestStudyHandler_GetSession_Forbidden(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	uid := uuid.New()
	sessionID := uuid.New()
	svc.EXPECT().GetSession(mock.Anything, uid, sessionID).Return(nil, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodGet, "/study/sessions/"+sessionID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- SubmitReview ---

func TestStudyHandler_SubmitReview_InvalidSessionID(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	body := `{"card_id":"` + uuid.New().String() + `","rating":3,"card":{"due":"2025-01-01T00:00:00Z","stability":1,"difficulty":5,"elapsed_days":0,"scheduled_days":1,"reps":1,"lapses":0,"state":1,"last_review":"2025-01-01T00:00:00Z"}}`
	req := httptest.NewRequest(http.MethodPost, "/study/sessions/bad-id/reviews", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStudyHandler_SubmitReview_InvalidBody(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	sessionID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/study/sessions/"+sessionID.String()+"/reviews", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStudyHandler_SubmitReview_ValidationError(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	sessionID := uuid.New()
	body := `{"rating":5}` // rating max is 4, missing card_id and card
	req := httptest.NewRequest(http.MethodPost, "/study/sessions/"+sessionID.String()+"/reviews", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- BatchReview ---

func TestStudyHandler_BatchReview_InvalidSessionID(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	body := `{"reviews":[]}`
	req := httptest.NewRequest(http.MethodPost, "/study/sessions/bad-id/reviews/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStudyHandler_BatchReview_InvalidBody(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	sessionID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/study/sessions/"+sessionID.String()+"/reviews/batch", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- EndSession ---

func TestStudyHandler_EndSession_Success(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	uid := uuid.New()
	sessionID := uuid.New()

	svc.EXPECT().EndSession(mock.Anything, uid, sessionID).Return(
		&domain.StudySession{ID: sessionID, UserID: uid}, nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/study/sessions/"+sessionID.String()+"/end", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStudyHandler_EndSession_InvalidID(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/study/sessions/bad-id/end", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Reminders ---

func TestStudyHandler_Reminders_Success(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	uid := uuid.New()

	svc.EXPECT().GetReminders(mock.Anything, uid, 24).Return(
		&dto.ReminderResponse{
			Decks:    []domain.DeckDueSummary{{DeckName: "Test Deck", DueNow: 5}},
			TotalDue: 5, Streak: 7, StudiedToday: true,
		}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/study/reminders", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStudyHandler_Reminders_CustomHours(t *testing.T) {
	svc := mockport.NewMockStudyServicer(t)
	h := NewStudyHandler(svc)
	router := setupStudyRouter(h)

	uid := uuid.New()

	svc.EXPECT().GetReminders(mock.Anything, uid, 48).Return(
		&dto.ReminderResponse{Decks: []domain.DeckDueSummary{}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/study/reminders?hours=48", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
