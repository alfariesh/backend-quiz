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

	// Quiz errors
	ErrAttemptCompleted  = errors.New("quiz attempt already completed")
	ErrQuestionNotInQuiz = errors.New("question does not belong to this quiz")
	ErrAlreadyAnswered   = errors.New("question already answered in this attempt")
	ErrQuizNotPublished  = errors.New("quiz is not published")
	ErrInsufficientCards = errors.New("not enough cards to generate quiz")

	// Auth errors
	ErrEmailNotVerified     = errors.New("email not verified")
	ErrEmailAlreadyVerified = errors.New("email already verified")
	ErrOTPInvalid           = errors.New("invalid or expired verification code")
	ErrOTPTooManyAttempts   = errors.New("too many verification attempts")
	ErrResetTokenInvalid    = errors.New("invalid or expired reset token")
	ErrRefreshTokenInvalid  = errors.New("invalid or expired refresh token")
	ErrRefreshTokenReuse    = errors.New("refresh token reuse detected")
	ErrAccountLocked        = errors.New("account temporarily locked due to too many failed attempts")
	ErrPasswordSameAsOld    = errors.New("new password must be different from the current password")
	ErrPasswordTooWeak      = errors.New("password does not meet complexity requirements")
	ErrPasswordCompromised  = errors.New("password has appeared in known data breaches — choose another")
	ErrOAuthStateInvalid    = errors.New("invalid oauth state")
	ErrSessionNotFound      = errors.New("session not found")
	ErrDeletionPending      = errors.New("account deletion already requested")
	ErrDeletionNotPending   = errors.New("no pending account deletion to cancel")
)
