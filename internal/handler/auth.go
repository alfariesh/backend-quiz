package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/dto"
	"github.com/alfariesh/backend-quiz/internal/middleware"
	"github.com/alfariesh/backend-quiz/internal/port"
	"github.com/alfariesh/backend-quiz/internal/service"
	"github.com/alfariesh/backend-quiz/pkg/oauth"
	"github.com/alfariesh/backend-quiz/pkg/validate"
)

const oauthStateCookie = "oauth_state"

type AuthHandler struct {
	authSvc          port.AuthServicer
	google           *oauth.GoogleClient
	successRedirect  string
	failureRedirect  string
	secureCookies    bool
	trustForwardedIP bool
}

type AuthHandlerOptions struct {
	Google           *oauth.GoogleClient
	SuccessRedirect  string
	FailureRedirect  string
	SecureCookies    bool
	TrustForwardedIP bool
}

func NewAuthHandler(authSvc port.AuthServicer, opts AuthHandlerOptions) *AuthHandler {
	return &AuthHandler{
		authSvc:          authSvc,
		google:           opts.Google,
		successRedirect:  opts.SuccessRedirect,
		failureRedirect:  opts.FailureRedirect,
		secureCookies:    opts.SecureCookies,
		trustForwardedIP: opts.TrustForwardedIP,
	}
}

// ── Registration & login ──────────────────────────────────────

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	meta := h.requestMeta(r)
	tokens, user, err := h.register(r, req, meta)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, map[string]any{"tokens": tokens, "user": user})
}

func (h *AuthHandler) register(r *http.Request, req dto.RegisterRequest, meta service.RequestMeta) (*dto.TokenPair, *domain.User, error) {
	if svc, ok := h.authSvc.(*service.AuthService); ok {
		return svc.RegisterWithMeta(r.Context(), req, meta)
	}
	return h.authSvc.Register(r.Context(), req)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	meta := h.requestMeta(r)
	tokens, user, err := h.login(r, req, meta)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{"tokens": tokens, "user": user})
}

func (h *AuthHandler) login(r *http.Request, req dto.LoginRequest, meta service.RequestMeta) (*dto.TokenPair, *domain.User, error) {
	if svc, ok := h.authSvc.(*service.AuthService); ok {
		return svc.LoginWithMeta(r.Context(), req, meta)
	}
	return h.authSvc.Login(r.Context(), req)
}

// ── Refresh & logout ──────────────────────────────────────────

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	meta := h.requestMeta(r)
	tokens, err := h.refresh(r, req.RefreshToken, meta)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) refresh(r *http.Request, token string, meta service.RequestMeta) (*dto.TokenPair, error) {
	if svc, ok := h.authSvc.(*service.AuthService); ok {
		return svc.RefreshTokenWithMeta(r.Context(), token, meta)
	}
	return h.authSvc.RefreshToken(r.Context(), token)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authSvc.Logout(r.Context(), req.RefreshToken); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "logged out")
}

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.authSvc.LogoutAll(r.Context(), userID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "all sessions revoked")
}

func (h *AuthHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sessions, err := h.authSvc.ListSessions(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}
	out := make([]dto.SessionDTO, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, dto.SessionDTO{
			ID:        s.ID.String(),
			UserAgent: s.UserAgent,
			IPAddress: s.IPAddress,
			CreatedAt: s.CreatedAt.Unix(),
			ExpiresAt: s.ExpiresAt.Unix(),
		})
	}
	JSON(w, http.StatusOK, out)
}

func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	if err := h.authSvc.RevokeSession(r.Context(), userID, sessionID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "session revoked")
}

// ── Account deletion (GDPR / app store) ───────────────────────

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	meta := h.requestMeta(r)
	user, err := h.requestDeletion(r, userID, meta)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) CancelAccountDeletion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	meta := h.requestMeta(r)
	user, err := h.cancelDeletion(r, userID, meta)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) ExportAccountData(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	meta := h.requestMeta(r)
	data, err := h.exportData(r, userID, meta)
	if err != nil {
		HandleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="account-export.json"`)
	w.WriteHeader(http.StatusOK)
	_ = writeJSON(w, data)
}

func (h *AuthHandler) requestDeletion(r *http.Request, userID uuid.UUID, meta service.RequestMeta) (*domain.User, error) {
	if svc, ok := h.authSvc.(*service.AuthService); ok {
		return svc.RequestAccountDeletion(r.Context(), userID, meta)
	}
	return h.authSvc.RequestAccountDeletionBasic(r.Context(), userID)
}

func (h *AuthHandler) cancelDeletion(r *http.Request, userID uuid.UUID, meta service.RequestMeta) (*domain.User, error) {
	if svc, ok := h.authSvc.(*service.AuthService); ok {
		return svc.CancelAccountDeletion(r.Context(), userID, meta)
	}
	return h.authSvc.CancelAccountDeletionBasic(r.Context(), userID)
}

func (h *AuthHandler) exportData(r *http.Request, userID uuid.UUID, meta service.RequestMeta) (map[string]any, error) {
	if svc, ok := h.authSvc.(*service.AuthService); ok {
		return svc.ExportAccountData(r.Context(), userID, meta)
	}
	return h.authSvc.ExportAccountDataBasic(r.Context(), userID)
}

// ── Profile ───────────────────────────────────────────────────

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

	var req dto.UpdateProfileRequest
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

// ── Email verification ────────────────────────────────────────

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyEmailRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.authSvc.VerifyEmail(r.Context(), req); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "email verified")
}

func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req dto.ResendVerificationRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.authSvc.ResendVerification(r.Context(), req.Email); err != nil {
		// ErrEmailAlreadyVerified is benign; surface as 400 via HandleError.
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "verification code sent")
}

// ── Password reset ────────────────────────────────────────────

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ForgotPasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.authSvc.ForgotPassword(r.Context(), req.Email); err != nil {
		HandleError(w, err)
		return
	}
	// Always respond success to avoid user enumeration.
	JSONMessage(w, http.StatusOK, "if the email exists, a reset link has been sent")
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.authSvc.ResetPassword(r.Context(), req); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "password reset successful")
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req dto.ChangePasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.authSvc.ChangePassword(r.Context(), userID, req); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "password changed")
}

// ── Google OAuth ──────────────────────────────────────────────

func (h *AuthHandler) GoogleRedirect(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		JSONError(w, http.StatusNotImplemented, "Google OAuth not configured")
		return
	}
	state, err := randomState()
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	http.Redirect(w, r, h.google.AuthURL(state), http.StatusFound)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		JSONError(w, http.StatusNotImplemented, "Google OAuth not configured")
		return
	}

	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || cookie.Value == "" {
		h.oauthFailure(w, r, "missing state cookie")
		return
	}
	// Clear state cookie (single-use).
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	q := r.URL.Query()
	if errParam := q.Get("error"); errParam != "" {
		h.oauthFailure(w, r, errParam)
		return
	}
	if q.Get("state") != cookie.Value {
		h.oauthFailure(w, r, "state mismatch")
		return
	}
	code := q.Get("code")
	if code == "" {
		h.oauthFailure(w, r, "missing code")
		return
	}

	accessToken, err := h.google.ExchangeCode(r.Context(), code)
	if err != nil {
		h.oauthFailure(w, r, "code exchange failed")
		return
	}
	ui, err := h.google.FetchUserInfo(r.Context(), accessToken)
	if err != nil {
		h.oauthFailure(w, r, "userinfo failed")
		return
	}
	if ui.Email == "" || !ui.EmailVerified {
		h.oauthFailure(w, r, "email not verified by provider")
		return
	}

	var avatarURL *string
	if ui.Picture != "" {
		p := ui.Picture
		avatarURL = &p
	}
	displayName := ui.Name
	if displayName == "" {
		displayName = ui.Email
	}

	meta := h.requestMeta(r)
	tokens, _, err := h.oauthLogin(r, "google", ui.Sub, ui.Email, displayName, avatarURL, meta)
	if err != nil {
		h.oauthFailure(w, r, "login failed")
		return
	}

	h.oauthSuccess(w, r, tokens)
}

func (h *AuthHandler) oauthLogin(r *http.Request, provider, providerID, email, displayName string, avatarURL *string, meta service.RequestMeta) (*dto.TokenPair, *domain.User, error) {
	if svc, ok := h.authSvc.(*service.AuthService); ok {
		return svc.FindOrCreateOAuthUserWithMeta(r.Context(), provider, providerID, email, displayName, avatarURL, meta)
	}
	return h.authSvc.FindOrCreateOAuthUser(r.Context(), provider, providerID, email, displayName, avatarURL)
}

func (h *AuthHandler) oauthSuccess(w http.ResponseWriter, r *http.Request, tokens *dto.TokenPair) {
	if h.successRedirect == "" {
		JSON(w, http.StatusOK, tokens)
		return
	}
	u, err := url.Parse(h.successRedirect)
	if err != nil {
		JSON(w, http.StatusOK, tokens)
		return
	}
	q := u.Query()
	q.Set("access_token", tokens.AccessToken)
	q.Set("refresh_token", tokens.RefreshToken)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *AuthHandler) oauthFailure(w http.ResponseWriter, r *http.Request, reason string) {
	if h.failureRedirect == "" {
		JSONError(w, http.StatusBadRequest, reason)
		return
	}
	u, err := url.Parse(h.failureRedirect)
	if err != nil {
		JSONError(w, http.StatusBadRequest, reason)
		return
	}
	q := u.Query()
	q.Set("error", reason)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// ── Helpers ───────────────────────────────────────────────────

func (h *AuthHandler) requestMeta(r *http.Request) service.RequestMeta {
	return service.RequestMeta{
		UserAgent: r.UserAgent(),
		IPAddress: middleware.ClientIP(r, h.trustForwardedIP),
	}
}

func randomState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

