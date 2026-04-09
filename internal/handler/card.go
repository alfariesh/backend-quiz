package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/port"
	"github.com/rekanesiads/backend-quiz/pkg/pagination"
	"github.com/rekanesiads/backend-quiz/pkg/validate"
)

type CardHandler struct {
	cardSvc port.CardServicer
}

func NewCardHandler(cardSvc port.CardServicer) *CardHandler {
	return &CardHandler{cardSvc: cardSvc}
}

func (h *CardHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	var req dto.CreateCardRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	card, err := h.cardSvc.Create(r.Context(), userID, deckID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, card)
}

func (h *CardHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	var req dto.BatchCreateRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	cards, err := h.cardSvc.BatchCreate(r.Context(), userID, deckID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, cards)
}

func (h *CardHandler) Get(w http.ResponseWriter, r *http.Request) {
	cardID, err := uuid.Parse(chi.URLParam(r, "cardID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	card, err := h.cardSvc.Get(r.Context(), cardID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, card)
}

func (h *CardHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	filter := domain.CardFilter{
		Tag:   r.URL.Query().Get("tag"),
		Query: r.URL.Query().Get("q"),
	}
	if stateStr := r.URL.Query().Get("state"); stateStr != "" {
		stateInt, err := strconv.Atoi(stateStr)
		if err == nil {
			state := domain.CardState(stateInt)
			filter.State = &state
		}
	}

	p := ParsePagination(r)
	cards, total, err := h.cardSvc.List(r.Context(), userID, deckID, filter, p.Limit, p.Offset)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, pagination.NewResponse(cards, total, p.Limit, p.Offset))
}

func (h *CardHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "cardID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	var req dto.UpdateCardRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	card, err := h.cardSvc.Update(r.Context(), userID, cardID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, card)
}

func (h *CardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "cardID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	if err := h.cardSvc.Delete(r.Context(), userID, cardID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "card deleted")
}

func (h *CardHandler) ResetFSRS(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "cardID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	card, err := h.cardSvc.ResetFSRS(r.Context(), userID, cardID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, card)
}

func (h *CardHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "cardID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	var req dto.SuspendRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.cardSvc.Suspend(r.Context(), userID, cardID, req.Suspended); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "card suspension updated")
}
