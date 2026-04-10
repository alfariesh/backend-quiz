package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/alfariesh/backend-quiz/internal/dto"
	"github.com/alfariesh/backend-quiz/internal/middleware"
	"github.com/alfariesh/backend-quiz/internal/port"
)

type StatsHandler struct {
	statsSvc port.StatsServicer
}

func NewStatsHandler(statsSvc port.StatsServicer) *StatsHandler {
	return &StatsHandler{statsSvc: statsSvc}
}

func (h *StatsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	stats, err := h.statsSvc.Overview(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) Heatmap(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	entries, err := h.statsSvc.Heatmap(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, entries)
}

func (h *StatsHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	forecast, err := h.statsSvc.Forecast(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, forecast)
}

func (h *StatsHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	entries, err := h.statsSvc.Leaderboard(r.Context(), limit)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, entries)
}

func (h *StatsHandler) DeckStats(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	stats, err := h.statsSvc.DeckStats(r.Context(), userID, deckID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) Mastery(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	mastery, err := h.statsSvc.Mastery(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}
	if mastery == nil {
		mastery = []dto.DeckMastery{}
	}
	JSON(w, http.StatusOK, mastery)
}

func (h *StatsHandler) WeakAreas(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	result, err := h.statsSvc.WeakAreas(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, result)
}

func (h *StatsHandler) TestComparison(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	comparison, err := h.statsSvc.TestComparison(r.Context(), userID, deckID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, comparison)
}
