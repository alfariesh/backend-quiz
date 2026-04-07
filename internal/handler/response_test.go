package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	JSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestJSONError(t *testing.T) {
	w := httptest.NewRecorder()

	JSONError(w, http.StatusBadRequest, "bad request")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "bad request", resp.Error)
}

func TestJSONMessage(t *testing.T) {
	w := httptest.NewRecorder()

	JSONMessage(w, http.StatusOK, "success")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Message)
}

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
		{"daily limit", domain.ErrDailyLimitReached, http.StatusTooManyRequests},
		{"unknown error", errors.New("something went wrong"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			HandleError(w, tt.err)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	body := strings.NewReader(`{"name": "test", "value": 42}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/json")

	var result struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	err := DecodeJSON(r, &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestDecodeJSON_UnknownFields(t *testing.T) {
	body := strings.NewReader(`{"name": "test", "unknown": true}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)

	var result struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(r, &result)
	assert.Error(t, err, "should reject unknown fields")
}

func TestParsePagination(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?limit=50&offset=10", nil)

	p := ParsePagination(r)
	assert.Equal(t, 50, p.Limit)
	assert.Equal(t, 10, p.Offset)
}

func TestParsePagination_Defaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	p := ParsePagination(r)
	assert.Equal(t, 20, p.Limit) // DefaultLimit
	assert.Equal(t, 0, p.Offset)
}
