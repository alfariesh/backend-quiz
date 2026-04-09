package domain

import "fmt"

// ErrorCode represents a machine-readable error classification.
type ErrorCode string

const (
	CodeNotFound           ErrorCode = "NOT_FOUND"
	CodeAlreadyExists      ErrorCode = "ALREADY_EXISTS"
	CodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	CodeForbidden          ErrorCode = "FORBIDDEN"
	CodeInvalidInput       ErrorCode = "INVALID_INPUT"
	CodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	CodeConflict           ErrorCode = "CONFLICT"
	CodeTooManyRequests    ErrorCode = "TOO_MANY_REQUESTS"
)

// AppError is a structured error that carries a machine-readable code,
// a human-readable message, and optional metadata for debugging.
// It wraps a sentinel error so errors.Is() continues to work.
type AppError struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Meta    map[string]any `json:"meta,omitempty"`
	Err     error          `json:"-"`
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// NewAppError creates a structured error wrapping a sentinel.
func NewAppError(sentinel error, code ErrorCode, msg string) *AppError {
	return &AppError{Code: code, Message: msg, Err: sentinel}
}

// NewAppErrorf creates a structured error with a formatted message.
func NewAppErrorf(sentinel error, code ErrorCode, format string, args ...any) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...), Err: sentinel}
}

// WithMeta attaches contextual metadata to an AppError.
func WithMeta(err *AppError, key string, value any) *AppError {
	if err.Meta == nil {
		err.Meta = make(map[string]any)
	}
	err.Meta[key] = value
	return err
}
