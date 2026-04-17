package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/pkg/pagination"
)

type APIResponse struct {
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Data: data})
}

func JSONMessage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Message: message})
}

func JSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Error: message})
}

func HandleError(w http.ResponseWriter, err error) {
	// Try structured AppError first
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		JSONError(w, codeToHTTPStatus(appErr.Code), appErr.Message)
		return
	}

	// Fallback to sentinel matching
	switch {
	case errors.Is(err, domain.ErrNotFound):
		JSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists),
		errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrDeckNameTaken):
		JSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrUnauthorized),
		errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrRefreshTokenInvalid),
		errors.Is(err, domain.ErrRefreshTokenReuse),
		errors.Is(err, domain.ErrResetTokenInvalid),
		errors.Is(err, domain.ErrOTPInvalid),
		errors.Is(err, domain.ErrOTPTooManyAttempts),
		errors.Is(err, domain.ErrEmailNotVerified):
		JSONError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrAccountLocked):
		JSONError(w, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyVerified),
		errors.Is(err, domain.ErrPasswordSameAsOld),
		errors.Is(err, domain.ErrPasswordTooWeak),
		errors.Is(err, domain.ErrPasswordCompromised),
		errors.Is(err, domain.ErrOAuthStateInvalid),
		errors.Is(err, domain.ErrDeletionPending),
		errors.Is(err, domain.ErrDeletionNotPending):
		JSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrSessionNotFound):
		JSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		JSONError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrCardSuspended),
		errors.Is(err, domain.ErrSessionEnded),
		errors.Is(err, domain.ErrFileTooLarge),
		errors.Is(err, domain.ErrUnsupportedMedia),
		errors.Is(err, domain.ErrAttemptCompleted),
		errors.Is(err, domain.ErrQuestionNotInQuiz),
		errors.Is(err, domain.ErrAlreadyAnswered),
		errors.Is(err, domain.ErrQuizNotPublished),
		errors.Is(err, domain.ErrInsufficientCards):
		JSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrDailyLimitReached),
		errors.Is(err, domain.ErrRAGDailyLimitReached):
		JSONError(w, http.StatusTooManyRequests, err.Error())
	default:
		JSONError(w, http.StatusInternalServerError, "internal server error")
	}
}

func codeToHTTPStatus(code domain.ErrorCode) int {
	switch code {
	case domain.CodeNotFound:
		return http.StatusNotFound
	case domain.CodeAlreadyExists, domain.CodeConflict:
		return http.StatusConflict
	case domain.CodeUnauthorized, domain.CodeInvalidCredentials:
		return http.StatusUnauthorized
	case domain.CodeForbidden:
		return http.StatusForbidden
	case domain.CodeInvalidInput:
		return http.StatusBadRequest
	case domain.CodeTooManyRequests:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// writeJSON encodes a raw value (no APIResponse envelope) to the writer.
// Used for file-attachment responses like the GDPR data export.
func writeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func ParsePagination(r *http.Request) pagination.Params {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return pagination.NewParams(limit, offset)
}
