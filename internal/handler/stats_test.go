package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	mockport "github.com/rekanesiads/backend-quiz/internal/mocks/port"
)

func setupStatsRouter(h *StatsHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/stats/overview", h.Overview)
	r.Get("/stats/heatmap", h.Heatmap)
	r.Get("/stats/forecast", h.Forecast)
	r.Get("/stats/leaderboard", h.Leaderboard)
	r.Get("/stats/decks/{deckID}", h.DeckStats)
	r.Get("/stats/mastery", h.Mastery)
	r.Get("/stats/weak-areas", h.WeakAreas)
	r.Get("/stats/decks/{deckID}/comparison", h.TestComparison)
	return r
}

// --- Overview ---

func TestStatsHandler_Overview_Success(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Overview(mock.Anything, uid).Return(
		&dto.OverviewStats{TotalReviews: 100, Streak: 7, RetentionRate: 0.85, TodayReviews: 10}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/overview", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStatsHandler_Overview_ServiceError(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Overview(mock.Anything, uid).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/stats/overview", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Heatmap ---

func TestStatsHandler_Heatmap_Success(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Heatmap(mock.Anything, uid).Return(
		[]dto.HeatmapEntry{{Date: "2025-01-01", Count: 5}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/heatmap", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStatsHandler_Heatmap_ServiceError(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Heatmap(mock.Anything, uid).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/stats/heatmap", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Forecast ---

func TestStatsHandler_Forecast_Success(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Forecast(mock.Anything, uid).Return(
		[]dto.ForecastDay{{Date: "2025-01-01", DueNew: 5, DueReview: 10}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/forecast", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStatsHandler_Forecast_ServiceError(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Forecast(mock.Anything, uid).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/stats/forecast", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Leaderboard ---

func TestStatsHandler_Leaderboard_Success(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	svc.EXPECT().Leaderboard(mock.Anything, 20).Return(
		[]domain.LeaderboardEntry{{DisplayName: "User1", Streak: 7}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/leaderboard", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStatsHandler_Leaderboard_CustomLimit(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	svc.EXPECT().Leaderboard(mock.Anything, 10).Return(
		[]domain.LeaderboardEntry{}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/leaderboard?limit=10", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- DeckStats ---

func TestStatsHandler_DeckStats_Success(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().DeckStats(mock.Anything, uid, deckID).Return(
		map[string]any{"total_cards": 10, "due_count": 3}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/decks/"+deckID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStatsHandler_DeckStats_InvalidID(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/stats/decks/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Mastery ---

func TestStatsHandler_Mastery_Success(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Mastery(mock.Anything, uid).Return(
		[]dto.DeckMastery{{DeckName: "D1", MasteryPercent: 80, MasteryLevel: "Advanced"}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/mastery", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStatsHandler_Mastery_Empty(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Mastery(mock.Anything, uid).Return([]dto.DeckMastery{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/stats/mastery", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "[]")
}

func TestStatsHandler_Mastery_ServiceError(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().Mastery(mock.Anything, uid).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/stats/mastery", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- WeakAreas ---

func TestStatsHandler_WeakAreas_Success(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().WeakAreas(mock.Anything, uid).Return(
		&dto.WeakAreasResponse{
			WeakAreas: []dto.WeakArea{{Tag: "tag1", WeakCards: 3}},
			WeakCards: []domain.Card{{Front: "Weak1"}},
		}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/weak-areas", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- TestComparison ---

func TestStatsHandler_TestComparison_Success(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().TestComparison(mock.Anything, uid, deckID).Return(
		&dto.TestComparison{DeckID: deckID, DeckName: "Test Deck"}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/stats/decks/"+deckID.String()+"/comparison", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStatsHandler_WeakAreas_ServiceError(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	uid := uuid.New()
	svc.EXPECT().WeakAreas(mock.Anything, uid).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/stats/weak-areas", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestStatsHandler_Leaderboard_ServiceError(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	svc.EXPECT().Leaderboard(mock.Anything, 20).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/stats/leaderboard", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestStatsHandler_TestComparison_InvalidDeckID(t *testing.T) {
	svc := mockport.NewMockStatsServicer(t)
	h := NewStatsHandler(svc)
	router := setupStatsRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/stats/decks/bad-id/comparison", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
