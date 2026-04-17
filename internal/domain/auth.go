package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EmailVerification struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	CodeHash   string
	Attempts   int
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

type PasswordResetToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ParentID  *uuid.UUID
	UserAgent string
	IPAddress string
	RevokedAt *time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
}

type LoginAttempt struct {
	ID         uuid.UUID
	Email      string
	IPAddress  string
	Successful bool
	CreatedAt  time.Time
}

type AuthAuditEvent string

const (
	AuditEventRegister               AuthAuditEvent = "register"
	AuditEventLoginSuccess           AuthAuditEvent = "login_success"
	AuditEventLoginFailed            AuthAuditEvent = "login_failed"
	AuditEventLogout                 AuthAuditEvent = "logout"
	AuditEventLogoutAll              AuthAuditEvent = "logout_all"
	AuditEventPasswordChanged        AuthAuditEvent = "password_changed"
	AuditEventPasswordResetRequested AuthAuditEvent = "password_reset_requested"
	AuditEventPasswordResetCompleted AuthAuditEvent = "password_reset_completed"
	AuditEventEmailVerified          AuthAuditEvent = "email_verified"
	AuditEventRefreshReuseDetected   AuthAuditEvent = "refresh_reuse_detected"
	AuditEventOAuthLogin             AuthAuditEvent = "oauth_login"
	AuditEventOAuthLinked            AuthAuditEvent = "oauth_linked"
	AuditEventAccountDeletionRequested AuthAuditEvent = "account_deletion_requested"
	AuditEventAccountDeletionCancelled AuthAuditEvent = "account_deletion_cancelled"
	AuditEventAccountDeleted          AuthAuditEvent = "account_deleted"
	AuditEventDataExported            AuthAuditEvent = "data_exported"
)

type AuthAuditLog struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	Event     AuthAuditEvent
	IPAddress string
	UserAgent string
	Metadata  map[string]any
	CreatedAt time.Time
}

type EmailVerificationRepository interface {
	Create(ctx context.Context, ev *EmailVerification) error
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*EmailVerification, error)
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	Consume(ctx context.Context, id uuid.UUID, at time.Time) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type PasswordResetTokenRepository interface {
	Create(ctx context.Context, t *PasswordResetToken) error
	GetByHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	Consume(ctx context.Context, id uuid.UUID, at time.Time) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, t *RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	GetByID(ctx context.Context, id uuid.UUID) (*RefreshToken, error)
	ListActiveByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]*RefreshToken, error)
	DeviceSeen(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (bool, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) error
	RevokeChildren(ctx context.Context, parentID uuid.UUID, at time.Time) error
}

type LoginAttemptRepository interface {
	Create(ctx context.Context, a *LoginAttempt) error
	CountRecentFailures(ctx context.Context, email string, since time.Time) (int, error)
}

type AuthAuditLogRepository interface {
	Create(ctx context.Context, log *AuthAuditLog) error
}
