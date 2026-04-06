package domain

import "errors"

var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken        = errors.New("email already taken")
	ErrDeckNameTaken     = errors.New("deck name already taken for this user")
	ErrSessionEnded      = errors.New("study session already ended")
	ErrCardNotInDeck     = errors.New("card does not belong to this deck")
	ErrCardSuspended     = errors.New("card is suspended")
	ErrDailyLimitReached = errors.New("daily limit reached")
	ErrFileTooLarge      = errors.New("file too large")
	ErrUnsupportedMedia  = errors.New("unsupported media type")
)
