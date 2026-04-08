package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/service"
	"github.com/rekanesiads/backend-quiz/pkg/validate"
)

type StudyHandler struct {
	studySvc *service.StudyService
}

func NewStudyHandler(studySvc *service.StudyService) *StudyHandler {
	return &StudyHandler{studySvc: studySvc}
}

func (h *StudyHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req service.StartSessionRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.studySvc.StartSession(r.Context(), userID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, resp)
}

func (h *StudyHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	session, err := h.studySvc.GetSession(r.Context(), userID, sessionID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, session)
}

func (h *StudyHandler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	var req service.SubmitReviewRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.studySvc.SubmitReview(r.Context(), userID, sessionID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, result)
}

func (h *StudyHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	session, err := h.studySvc.EndSession(r.Context(), userID, sessionID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, session)
}

