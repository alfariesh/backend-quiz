package handler

import (
	"net/http"

	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/service"
	"github.com/rekanesiads/backend-quiz/pkg/validate"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, user, err := h.authSvc.Register(r.Context(), req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, map[string]any{
		"tokens": tokens,
		"user":   user,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, user, err := h.authSvc.Login(r.Context(), req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"tokens": tokens,
		"user":   user,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.authSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.authSvc.GetProfile(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.UpdateProfileRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authSvc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) GoogleRedirect(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement Google OAuth redirect
	JSONError(w, http.StatusNotImplemented, "Google OAuth not configured")
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement Google OAuth callback
	JSONError(w, http.StatusNotImplemented, "Google OAuth not configured")
}
