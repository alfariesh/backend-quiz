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
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
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

	CreateOAuthAccount(ctx context.Context, account *OAuthAccount) error
	GetOAuthAccount(ctx context.Context, provider, providerID string) (*OAuthAccount, error)
}
