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

func setupDeckRouter(h *DeckHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/decks", h.Create)
	r.Get("/decks", h.List)
	r.Get("/decks/public", h.ListPublic)
	r.Post("/decks/import", h.Import)
	r.Post("/decks/clone/{shareCode}", h.Clone)
	r.Get("/decks/{deckID}", h.Get)
	r.Put("/decks/{deckID}", h.Update)
	r.Delete("/decks/{deckID}", h.Delete)
	r.Get("/decks/{deckID}/export", h.Export)
	r.Post("/decks/{deckID}/share", h.Share)
	r.Delete("/decks/{deckID}/share", h.Unshare)
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

func TestDeckHandler_Create_ServiceError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	svc.EXPECT().Create(mock.Anything, uid, mock.Anything).Return(nil, assert.AnError)

	body := `{"name":"Test Deck"}`
	req := httptest.NewRequest(http.MethodPost, "/decks", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
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

// --- Share ---

func TestDeckHandler_Share_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Share(mock.Anything, uid, deckID, mock.Anything).Return(
		&domain.DeckShare{DeckID: deckID, ShareCode: "abc123", IsPublic: true}, nil,
	)

	body := `{"is_public":true}`
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/share", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestDeckHandler_Share_InvalidID(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/decks/bad-id/share", strings.NewReader(`{}`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Share_InvalidBody(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	deckID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/share", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Share_Forbidden(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Share(mock.Anything, uid, deckID, mock.Anything).Return(nil, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodPost, "/decks/"+deckID.String()+"/share", strings.NewReader(`{}`))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- Unshare ---

func TestDeckHandler_Unshare_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Unshare(mock.Anything, uid, deckID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/decks/"+deckID.String()+"/share", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "share removed")
}

func TestDeckHandler_Unshare_InvalidID(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/decks/bad-id/share", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Clone ---

func TestDeckHandler_Clone_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	svc.EXPECT().Clone(mock.Anything, uid, "abc123").Return(
		&domain.Deck{ID: uuid.New(), UserID: uid, Name: "Cloned Deck"}, nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/decks/clone/abc123", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, "Cloned Deck", data["name"])
}

func TestDeckHandler_Clone_NotFound(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	svc.EXPECT().Clone(mock.Anything, uid, "invalid").Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodPost, "/decks/clone/invalid", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- ListPublic ---

func TestDeckHandler_ListPublic_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	svc.EXPECT().ListPublic(mock.Anything, "quran", 20, 0).Return(
		[]domain.DeckWithCounts{{Deck: domain.Deck{Name: "Quran Deck"}}}, 1, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/decks/public?q=quran", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeckHandler_ListPublic_ServiceError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	svc.EXPECT().ListPublic(mock.Anything, "", 20, 0).Return(nil, 0, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/decks/public", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Export ---

func TestDeckHandler_Export_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	exportData := []byte(`{"name":"My Deck","cards":[]}`)
	svc.EXPECT().Export(mock.Anything, uid, deckID).Return(exportData, nil)

	req := httptest.NewRequest(http.MethodGet, "/decks/"+deckID.String()+"/export", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
	assert.JSONEq(t, `{"name":"My Deck","cards":[]}`, rec.Body.String())
}

func TestDeckHandler_Export_InvalidID(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/decks/bad-id/export", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Export_Forbidden(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Export(mock.Anything, uid, deckID).Return(nil, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodGet, "/decks/"+deckID.String()+"/export", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- Import ---

func TestDeckHandler_Import_Success(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	svc.EXPECT().Import(mock.Anything, uid, mock.Anything).Return(
		&domain.Deck{ID: uuid.New(), UserID: uid, Name: "Imported"}, nil,
	)

	body := `{"name":"Imported","cards":[{"front":"Q","back":"A"}]}`
	req := httptest.NewRequest(http.MethodPost, "/decks/import", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, "Imported", data["name"])
}

func TestDeckHandler_Import_InvalidBody(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/decks/import", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Import_ValidationError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	body := `{"cards":[{"front":"Q","back":"A"}]}`
	req := httptest.NewRequest(http.MethodPost, "/decks/import", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Update_InvalidID(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/decks/not-a-uuid", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Update_InvalidBody(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	deckID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/decks/"+deckID.String(), strings.NewReader(`not json`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeckHandler_Update_ServiceError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Update(mock.Anything, uid, deckID, mock.Anything).Return(nil, domain.ErrNotFound)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/decks/"+deckID.String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeckHandler_List_ServiceError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	svc.EXPECT().List(mock.Anything, uid, mock.Anything, mock.Anything).Return(nil, 0, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/decks", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDeckHandler_Delete_ServiceError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Delete(mock.Anything, uid, deckID).Return(domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodDelete, "/decks/"+deckID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeckHandler_Unshare_ServiceError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	deckID := uuid.New()
	svc.EXPECT().Unshare(mock.Anything, uid, deckID).Return(domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodDelete, "/decks/"+deckID.String()+"/share", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeckHandler_Import_ServiceError(t *testing.T) {
	svc := mockport.NewMockDeckServicer(t)
	h := NewDeckHandler(svc)
	router := setupDeckRouter(h)

	uid := uuid.New()
	svc.EXPECT().Import(mock.Anything, uid, mock.Anything).Return(nil, assert.AnError)

	body := `{"name":"Test","cards":[{"front":"Q","back":"A"}]}`
	req := httptest.NewRequest(http.MethodPost, "/decks/import", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
