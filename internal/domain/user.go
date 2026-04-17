package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID  `json:"id"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"`
	DisplayName      string     `json:"display_name"`
	Timezone         string     `json:"timezone"`
	DesiredRetention float64    `json:"desired_retention"`
	DailyNewLimit    int        `json:"daily_new_limit"`
	DailyReviewLimit int        `json:"daily_review_limit"`
	FSRSWeights      []float64  `json:"fsrs_weights,omitempty"`
	ReminderEnabled  bool       `json:"reminder_enabled"`
	ReminderTime     string     `json:"reminder_time"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	DeletionRequestedAt *time.Time `json:"deletion_requested_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (u *User) IsEmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

func (u *User) IsPendingDeletion() bool {
	return u.DeletionRequestedAt != nil
}

type UserDataExportRepository interface {
	Dump(ctx context.Context, userID uuid.UUID) (map[string]any, error)
}

type OAuthAccount struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	Provider   string    `json:"provider"`
	ProviderID string    `json:"provider_id"`
	Email      string    `json:"email"`
	AvatarURL  *string   `json:"avatar_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error

	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	MarkEmailVerified(ctx context.Context, userID uuid.UUID, at time.Time) error

	RequestDeletion(ctx context.Context, userID uuid.UUID, at time.Time) error
	CancelDeletion(ctx context.Context, userID uuid.UUID) error
	ListExpiredDeletions(ctx context.Context, before time.Time, limit int) ([]uuid.UUID, error)

	CreateOAuthAccount(ctx context.Context, account *OAuthAccount) error
	GetOAuthAccount(ctx context.Context, provider, providerID string) (*OAuthAccount, error)
	ListOAuthAccountsByUser(ctx context.Context, userID uuid.UUID) ([]OAuthAccount, error)
}
