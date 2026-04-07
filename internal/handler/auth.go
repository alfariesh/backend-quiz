package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/service"
	"github.com/rekanesiads/backend-quiz/pkg/validate"
)

type AuthHandler struct {
	authSvc  *service.AuthService
	oauthSvc *service.OAuthService
}

func NewAuthHandler(authSvc *service.AuthService, oauthSvc *service.OAuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, oauthSvc: oauthSvc}
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
	b := make([]byte, 16)
	rand.Read(b)
	state := hex.EncodeToString(b)

	// In production, store state in a secure cookie or session for CSRF validation
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	url := h.oauthSvc.GetAuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state parameter for CSRF protection
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		JSONError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		JSONError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	// Exchange code for Google user info
	googleUser, err := h.oauthSvc.ExchangeCode(r.Context(), code)
	if err != nil {
		JSONError(w, http.StatusBadGateway, "failed to authenticate with Google")
		return
	}

	// Find or create user
	var avatarURL *string
	if googleUser.Picture != "" {
		avatarURL = &googleUser.Picture
	}

	tokens, user, err := h.authSvc.FindOrCreateOAuthUser(
		r.Context(),
		"google",
		googleUser.ID,
		googleUser.Email,
		googleUser.Name,
		avatarURL,
	)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"tokens": tokens,
		"user":   user,
	})
}
