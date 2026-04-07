package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/pkg/pagination"
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
	switch {
	case errors.Is(err, domain.ErrNotFound):
		JSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists),
		errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrDeckNameTaken):
		JSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrUnauthorized),
		errors.Is(err, domain.ErrInvalidCredentials):
		JSONError(w, http.StatusUnauthorized, err.Error())
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
	case errors.Is(err, domain.ErrDailyLimitReached):
		JSONError(w, http.StatusTooManyRequests, err.Error())
	default:
		JSONError(w, http.StatusInternalServerError, "internal server error")
	}
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
