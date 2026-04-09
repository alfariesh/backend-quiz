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

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	mockport "github.com/rekanesiads/backend-quiz/internal/mocks/port"
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

func userID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}
