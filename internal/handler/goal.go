package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/service"
	"github.com/rekanesiads/backend-quiz/pkg/validate"
)

type GoalHandler struct {
	goalSvc *service.GoalService
}

func NewGoalHandler(goalSvc *service.GoalService) *GoalHandler {
	return &GoalHandler{goalSvc: goalSvc}
}

func (h *GoalHandler) SetGoal(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req service.SetGoalRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	goal, err := h.goalSvc.SetGoal(r.Context(), userID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, goal)
}

func (h *GoalHandler) ListWithProgress(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	progress, err := h.goalSvc.ListWithProgress(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}
	if progress == nil {
		progress = []service.GoalProgress{}
	}
	JSON(w, http.StatusOK, progress)
}

func (h *GoalHandler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	goalID, err := uuid.Parse(chi.URLParam(r, "goalID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid goal id")
		return
	}

	if err := h.goalSvc.DeleteGoal(r.Context(), userID, goalID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "goal deleted")
}
