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

	"github.com/alfariesh/backend-quiz/internal/domain"
	mockport "github.com/alfariesh/backend-quiz/internal/mocks/port"
)

func setupCardRouter(h *CardHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/decks/{deckID}/cards", h.Create)
	r.Post("/decks/{deckID}/cards/batch", h.BatchCreate)
	r.Get("/decks/{deckID}/cards", h.List)
	r.Get("/cards/{cardID}", h.Get)
	r.Put("/cards/{cardID}", h.Update)
	r.Delete("/cards/{cardID}", h.Delete)
	r.Post("/cards/{cardID}/reset", h.ResetFSRS)
	r.Put("/cards/{cardID}/suspend", h.Suspend)
	return r
}

// --- Create ---

func TestCardHandler_Create_Success(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().Create(mock.Anything, uid, deckID, mock.Anything).Return(
		&domain.Card{ID: uuid.New(), DeckID: deckID, Front: "Q1", Back: "A1"}, nil,
	)

	body := `{"front":"Q1","back":"A1"}`
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/cards", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, "Q1", data["front"])
}

func TestCardHandler_Create_InvalidDeckID(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	body := `{"front":"Q","back":"A"}`
	req := httptest.NewRequest(http.MethodPost, "/decks/not-uuid/cards", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCardHandler_Create_InvalidBody(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	deckID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/cards", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCardHandler_Create_ValidationError(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	deckID := uuid.New()
	body := `{"front":""}` // missing back, empty front
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/cards", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCardHandler_Create_Forbidden(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	deckID := uuid.New()
	svc.EXPECT().Create(mock.Anything, mock.Anything, deckID, mock.Anything).Return(nil, domain.ErrForbidden)

	body := `{"front":"Q","back":"A"}`
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/cards", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- BatchCreate ---

func TestCardHandler_BatchCreate_Success(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().BatchCreate(mock.Anything, uid, deckID, mock.Anything).Return(
		[]*domain.Card{{Front: "Q1"}, {Front: "Q2"}}, nil,
	)

	body := `{"cards":[{"front":"Q1","back":"A1"},{"front":"Q2","back":"A2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/cards/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestCardHandler_BatchCreate_InvalidBody(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	deckID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/cards/batch", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCardHandler_BatchCreate_ValidationError(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	deckID := uuid.New()
	body := `{"cards":[]}` // min=1
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/cards/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Get ---

func TestCardHandler_Get_Success(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	cardID := uuid.New()
	svc.EXPECT().Get(mock.Anything, cardID).Return(&domain.Card{ID: cardID, Front: "Q", Back: "A"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/cards/"+cardID.String(), nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCardHandler_Get_InvalidID(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/cards/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCardHandler_Get_NotFound(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	cardID := uuid.New()
	svc.EXPECT().Get(mock.Anything, cardID).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/cards/"+cardID.String(), nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- List ---

func TestCardHandler_List_Success(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().List(mock.Anything, uid, deckID, mock.Anything, 20, 0).Return(
		[]domain.Card{{Front: "Q"}}, 1, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/decks/"+deckID.String()+"/cards", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCardHandler_List_WithFilters(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().List(mock.Anything, uid, deckID, mock.Anything, 10, 5).Return(
		[]domain.Card{{Front: "Q"}}, 1, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/decks/"+deckID.String()+"/cards?tag=quran&q=surah&state=1&limit=10&offset=5", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCardHandler_List_ServiceError(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().List(mock.Anything, uid, deckID, mock.Anything, 20, 0).Return(nil, 0, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodGet, "/decks/"+deckID.String()+"/cards", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCardHandler_List_InvalidDeckID(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/decks/bad-id/cards", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Update ---

func TestCardHandler_Update_Success(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	cardID := uuid.New()

	svc.EXPECT().Update(mock.Anything, uid, cardID, mock.Anything).Return(
		&domain.Card{ID: cardID, Front: "New"}, nil,
	)

	body := `{"front":"New"}`
	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID.String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCardHandler_Update_InvalidID(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	body := `{"front":"New"}`
	req := httptest.NewRequest(http.MethodPut, "/cards/bad-id", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCardHandler_Update_InvalidBody(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	cardID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID.String(), strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Delete ---

func TestCardHandler_Delete_Success(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	cardID := uuid.New()

	svc.EXPECT().Delete(mock.Anything, uid, cardID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/cards/"+cardID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "card deleted")
}

func TestCardHandler_Delete_InvalidID(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/cards/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ResetFSRS ---

func TestCardHandler_ResetFSRS_Success(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	cardID := uuid.New()

	svc.EXPECT().ResetFSRS(mock.Anything, uid, cardID).Return(&domain.Card{ID: cardID}, nil)

	req := httptest.NewRequest(http.MethodPost, "/cards/"+cardID.String()+"/reset", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCardHandler_ResetFSRS_InvalidID(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/cards/bad-id/reset", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Suspend ---

func TestCardHandler_Suspend_Success(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	cardID := uuid.New()

	svc.EXPECT().Suspend(mock.Anything, uid, cardID, true).Return(nil)

	body := `{"suspended":true}`
	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID.String()+"/suspend", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "card suspension updated")
}

func TestCardHandler_Suspend_InvalidID(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	body := `{"suspended":true}`
	req := httptest.NewRequest(http.MethodPut, "/cards/bad-id/suspend", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCardHandler_Update_ServiceError(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	cardID := uuid.New()
	svc.EXPECT().Update(mock.Anything, uid, cardID, mock.Anything).Return(nil, domain.ErrForbidden)

	body := `{"front":"New"}`
	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID.String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCardHandler_Delete_ServiceError(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	cardID := uuid.New()
	svc.EXPECT().Delete(mock.Anything, uid, cardID).Return(domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodDelete, "/cards/"+cardID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCardHandler_ResetFSRS_ServiceError(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	cardID := uuid.New()
	svc.EXPECT().ResetFSRS(mock.Anything, uid, cardID).Return(nil, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodPost, "/cards/"+cardID.String()+"/reset", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCardHandler_Suspend_ServiceError(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	cardID := uuid.New()
	svc.EXPECT().Suspend(mock.Anything, uid, cardID, true).Return(domain.ErrForbidden)

	body := `{"suspended":true}`
	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID.String()+"/suspend", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCardHandler_BatchCreate_InvalidDeckID(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	body := `{"cards":[{"front":"Q","back":"A"}]}`
	req := httptest.NewRequest(http.MethodPost, "/decks/bad-id/cards/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCardHandler_BatchCreate_ServiceError(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().BatchCreate(mock.Anything, uid, deckID, mock.Anything).Return(nil, domain.ErrForbidden)

	body := `{"cards":[{"front":"Q","back":"A"}]}`
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/cards/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCardHandler_Suspend_InvalidBody(t *testing.T) {
	svc := mockport.NewMockCardServicer(t)
	h := NewCardHandler(svc)
	router := setupCardRouter(h)

	cardID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID.String()+"/suspend", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
