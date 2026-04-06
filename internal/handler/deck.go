package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/service"
	"github.com/rekanesiads/backend-quiz/pkg/pagination"
	"github.com/rekanesiads/backend-quiz/pkg/validate"
)

type DeckHandler struct {
	deckSvc *service.DeckService
}

func NewDeckHandler(deckSvc *service.DeckService) *DeckHandler {
	return &DeckHandler{deckSvc: deckSvc}
}

func (h *DeckHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req service.CreateDeckRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	deck, err := h.deckSvc.Create(r.Context(), userID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, deck)
}

func (h *DeckHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	deck, err := h.deckSvc.Get(r.Context(), userID, deckID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, deck)
}

func (h *DeckHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	p := ParsePagination(r)

	decks, total, err := h.deckSvc.List(r.Context(), userID, p.Limit, p.Offset)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, pagination.NewResponse(decks, total, p.Limit, p.Offset))
}

func (h *DeckHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	var req service.UpdateDeckRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	deck, err := h.deckSvc.Update(r.Context(), userID, deckID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, deck)
}

func (h *DeckHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	if err := h.deckSvc.Delete(r.Context(), userID, deckID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "deck deleted")
}

func (h *DeckHandler) Share(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	var req service.ShareDeckRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	share, err := h.deckSvc.Share(r.Context(), userID, deckID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, share)
}

func (h *DeckHandler) Unshare(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	if err := h.deckSvc.Unshare(r.Context(), userID, deckID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "share removed")
}

func (h *DeckHandler) Clone(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	shareCode := chi.URLParam(r, "shareCode")

	deck, err := h.deckSvc.Clone(r.Context(), userID, shareCode)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, deck)
}

func (h *DeckHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	p := ParsePagination(r)

	decks, total, err := h.deckSvc.ListPublic(r.Context(), search, p.Limit, p.Offset)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, pagination.NewResponse(decks, total, p.Limit, p.Offset))
}

func (h *DeckHandler) Export(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	data, err := h.deckSvc.Export(r.Context(), userID, deckID)
	if err != nil {
		HandleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=deck-export.json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *DeckHandler) Import(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req service.ImportDeckRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	deck, err := h.deckSvc.Import(r.Context(), userID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, deck)
}
