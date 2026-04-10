package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

// --- JSON ---

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusOK, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, "value", data["key"])
}

func TestJSON_Created(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusCreated, "created")

	assert.Equal(t, http.StatusCreated, rec.Code)
}

// --- JSONMessage ---

func TestJSONMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	JSONMessage(rec, http.StatusOK, "deck deleted")

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "deck deleted", resp.Message)
	assert.Empty(t, resp.Error)
}

// --- JSONError ---

func TestJSONError(t *testing.T) {
	rec := httptest.NewRecorder()
	JSONError(rec, http.StatusBadRequest, "invalid input")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "invalid input", resp.Error)
	assert.Nil(t, resp.Data)
}

// --- HandleError ---

func TestHandleError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"email taken", domain.ErrEmailTaken, http.StatusConflict},
		{"already exists", domain.ErrAlreadyExists, http.StatusConflict},
		{"unauthorized", domain.ErrUnauthorized, http.StatusUnauthorized},
		{"invalid credentials", domain.ErrInvalidCredentials, http.StatusUnauthorized},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden},
		{"invalid input", domain.ErrInvalidInput, http.StatusBadRequest},
		{"card suspended", domain.ErrCardSuspended, http.StatusBadRequest},
		{"session ended", domain.ErrSessionEnded, http.StatusBadRequest},
		{"file too large", domain.ErrFileTooLarge, http.StatusBadRequest},
		{"unsupported media", domain.ErrUnsupportedMedia, http.StatusBadRequest},
		{"attempt completed", domain.ErrAttemptCompleted, http.StatusBadRequest},
		{"question not in quiz", domain.ErrQuestionNotInQuiz, http.StatusBadRequest},
		{"already answered", domain.ErrAlreadyAnswered, http.StatusBadRequest},
		{"quiz not published", domain.ErrQuizNotPublished, http.StatusBadRequest},
		{"insufficient cards", domain.ErrInsufficientCards, http.StatusBadRequest},
		{"daily limit", domain.ErrDailyLimitReached, http.StatusTooManyRequests},
		{"unknown error", assert.AnError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			HandleError(rec, tt.err)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestHandleError_AppError(t *testing.T) {
	tests := []struct {
		name       string
		code       domain.ErrorCode
		wantStatus int
	}{
		{"not found", domain.CodeNotFound, http.StatusNotFound},
		{"already exists", domain.CodeAlreadyExists, http.StatusConflict},
		{"conflict", domain.CodeConflict, http.StatusConflict},
		{"unauthorized", domain.CodeUnauthorized, http.StatusUnauthorized},
		{"invalid credentials", domain.CodeInvalidCredentials, http.StatusUnauthorized},
		{"forbidden", domain.CodeForbidden, http.StatusForbidden},
		{"invalid input", domain.CodeInvalidInput, http.StatusBadRequest},
		{"too many requests", domain.CodeTooManyRequests, http.StatusTooManyRequests},
		{"unknown code", domain.ErrorCode("UNKNOWN"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := domain.NewAppError(domain.ErrInvalidInput, tt.code, "test message")
			rec := httptest.NewRecorder()
			HandleError(rec, appErr)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var resp APIResponse
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Equal(t, "test message", resp.Error)
		})
	}
}

// --- DecodeJSON ---

func TestDecodeJSON_Success(t *testing.T) {
	body := strings.NewReader(`{"name":"test","value":42}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)

	var dst struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	err := DecodeJSON(req, &dst)

	require.NoError(t, err)
	assert.Equal(t, "test", dst.Name)
	assert.Equal(t, 42, dst.Value)
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/", body)

	var dst struct{ Name string }
	err := DecodeJSON(req, &dst)
	assert.Error(t, err)
}

func TestDecodeJSON_UnknownFields(t *testing.T) {
	body := strings.NewReader(`{"name":"test","unknown_field":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(req, &dst)
	assert.Error(t, err) // DisallowUnknownFields
}

// --- ParsePagination ---

func TestParsePagination_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	p := ParsePagination(req)

	assert.Equal(t, 20, p.Limit)  // default
	assert.Equal(t, 0, p.Offset)
}

func TestParsePagination_CustomValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=50&offset=10", nil)
	p := ParsePagination(req)

	assert.Equal(t, 50, p.Limit)
	assert.Equal(t, 10, p.Offset)
}

func TestParsePagination_InvalidValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=abc&offset=-5", nil)
	p := ParsePagination(req)

	assert.Equal(t, 20, p.Limit) // fallback to default
	assert.Equal(t, 0, p.Offset) // negative clamped to 0
}
