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
	mockport "github.com/rekanesiads/backend-quiz/internal/mocks/port"
)

func setupDeckRouter(h *DeckHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/decks", h.Create)
	r.Get("/decks", h.List)
	r.Get("/decks/{deckID}", h.Get)
	r.Put("/decks/{deckID}", h.Update)
	r.Delete("/decks/{deckID}", h.Delete)
	return r
}

func TestDeckHandler_Create_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	svc.EXPECT().Create(mock.Anything, uid, mock.Anything).Return(
		&domain.Deck{ID: uuid.New(), UserID: uid, Name: "My Deck", Description: "test"}, nil,
	)

	body := `{"name":"My Deck","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/decks", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, "My Deck", data["name"])
}

func TestDeckHandler_Create_InvalidBody(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/decks", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Create_ValidationError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	body := `{"description":"no name"}`
	req := httptest.NewRequest(http.MethodPost, "/decks", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Get_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Get(mock.Anything, uid, deckID).Return(
		&domain.Deck{ID: deckID, UserID: uid, Name: "Test Deck"}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/decks/"+deckID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeckHandler_Get_InvalidID(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/decks/not-a-uuid", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Get_NotFound(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Get(mock.Anything, uid, deckID).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/decks/"+deckID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeckHandler_Get_Forbidden(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Get(mock.Anything, uid, deckID).Return(nil, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodGet, "/decks/"+deckID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeckHandler_List_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	svc.EXPECT().List(mock.Anything, uid, 20, 0).Return(
		[]domain.DeckWithCounts{{Deck: domain.Deck{Name: "D1"}}}, 1, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/decks", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeckHandler_Update_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().Update(mock.Anything, uid, deckID, mock.Anything).Return(
		&domain.Deck{ID: deckID, UserID: uid, Name: "New Name"}, nil,
	)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/decks/"+deckID.String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeckHandler_Delete_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().Delete(mock.Anything, uid, deckID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/decks/"+deckID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "deck deleted")
}

func TestDeckHandler_Delete_InvalidID(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/decks/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
