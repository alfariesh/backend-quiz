package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/dto"
	mockport "github.com/alfariesh/backend-quiz/internal/mocks/port"
)

func TestAuthHandler_Register_Success(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	uid := uuid.New()
	svc.EXPECT().Register(mock.Anything, mock.Anything).Return(
		&dto.TokenPair{AccessToken: "at", RefreshToken: "rt", ExpiresAt: 123},
		&domain.User{ID: uid, Email: "test@example.com", DisplayName: "Test User"},
		nil,
	)

	body := `{"email":"test@example.com","password":"password123","display_name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Contains(t, data, "tokens")
	assert.Contains(t, data, "user")
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`not json`))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	body := `{"email":"not-an-email"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Register_EmailTaken(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	svc.EXPECT().Register(mock.Anything, mock.Anything).Return(nil, nil, domain.ErrEmailTaken)

	body := `{"email":"taken@example.com","password":"password123","display_name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	uid := uuid.New()
	svc.EXPECT().Login(mock.Anything, mock.Anything).Return(
		&dto.TokenPair{AccessToken: "at", RefreshToken: "rt", ExpiresAt: 123},
		&domain.User{ID: uid, Email: "login@example.com"},
		nil,
	)

	body := `{"email":"login@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Contains(t, data, "tokens")
}

func TestAuthHandler_Login_InvalidBody(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{bad}`))
	rec := httptest.NewRecorder()

	h.Login(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Login_ValidationError(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	body := `{"email":"not-an-email"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Login_NotFound(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	svc.EXPECT().Login(mock.Anything, mock.Anything).Return(nil, nil, domain.ErrInvalidCredentials)

	body := `{"email":"nope@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthHandler_Me_Success(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	uid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user := &domain.User{ID: uid, DisplayName: "Test User", Email: "test@example.com"}
	svc.EXPECT().GetProfile(mock.Anything, uid).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthHandler_Me_NoUserID(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	rec := httptest.NewRecorder()

	h.Me(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	svc.EXPECT().RefreshToken(mock.Anything, "valid-refresh-token").Return(
		&dto.TokenPair{AccessToken: "new-at", RefreshToken: "new-rt", ExpiresAt: 456},
		nil,
	)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthHandler_Refresh_InvalidBody(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`bad`))
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- UpdateProfile ---

func TestAuthHandler_UpdateProfile_Success(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	uid := uuid.New()
	displayName := "New Name"
	svc.EXPECT().UpdateProfile(mock.Anything, uid, mock.Anything).Return(
		&domain.User{ID: uid, DisplayName: displayName, Email: "test@example.com"}, nil,
	)

	body := `{"display_name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/auth/me", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, "New Name", data["display_name"])
}

func TestAuthHandler_UpdateProfile_NoUserID(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	body := `{"display_name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/auth/me", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthHandler_UpdateProfile_InvalidBody(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/auth/me", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_UpdateProfile_ValidationError(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	// display_name max is 100, send 101+ chars
	body := `{"display_name":"` + strings.Repeat("x", 101) + `"}`
	req := httptest.NewRequest(http.MethodPut, "/auth/me", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_UpdateProfile_ServiceError(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	uid := uuid.New()
	svc.EXPECT().UpdateProfile(mock.Anything, uid, mock.Anything).Return(nil, domain.ErrNotFound)

	body := `{"display_name":"Test"}`
	req := httptest.NewRequest(http.MethodPut, "/auth/me", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- GoogleRedirect & GoogleCallback ---

func TestAuthHandler_GoogleRedirect_NotImplemented(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	rec := httptest.NewRecorder()

	h.GoogleRedirect(rec, req)
	assert.Equal(t, http.StatusNotImplemented, rec.Code)
}

func TestAuthHandler_GoogleCallback_NotImplemented(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil)
	rec := httptest.NewRecorder()

	h.GoogleCallback(rec, req)
	assert.Equal(t, http.StatusNotImplemented, rec.Code)
}

// --- Refresh: service error ---

func TestAuthHandler_Refresh_ServiceError(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	svc.EXPECT().RefreshToken(mock.Anything, "bad-token").Return(nil, domain.ErrUnauthorized)

	body := `{"refresh_token":"bad-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Me: service error ---

func TestAuthHandler_Me_ServiceError(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	uid := uuid.New()
	svc.EXPECT().GetProfile(mock.Anything, uid).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	h.Me(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- Login: service error ---

func TestAuthHandler_Login_ServiceError(t *testing.T) {
	svc := mockport.NewMockAuthServicer(t)
	h := NewAuthHandler(svc)

	svc.EXPECT().Login(mock.Anything, mock.Anything).Return(nil, nil, domain.ErrInvalidCredentials)

	body := `{"email":"test@example.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
